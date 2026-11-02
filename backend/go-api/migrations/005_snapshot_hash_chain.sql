-- ============================================================
-- Snapshot hash chain
--
-- When a snapshot becomes completed, the database stamps it with
-- a SHA-256 record_hash of its contents plus the previous
-- snapshot's hash. Changing any stored value breaks the chain,
-- and verify_snapshot_chain() reports where.
--
-- Everything is done here so there is one implementation,
-- whichever service (Go, Python, or a person in psql) writes.
-- ============================================================


-- ============================================================
-- Chain columns
--
-- Empty until the snapshot completes; set only by the trigger.
-- ============================================================
ALTER TABLE snapshots
    ADD COLUMN chain_seq        BIGINT UNIQUE,
    ADD COLUMN prev_record_hash TEXT,
    ADD COLUMN record_hash      TEXT UNIQUE;


-- ============================================================
-- Chain head
--
-- One row holding the latest position and hash. Completing a
-- snapshot locks this row, so two snapshots finishing at the
-- same moment cannot both link to the same previous hash.
-- ============================================================
CREATE TABLE snapshot_chain_head (
    id   BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id), -- allows only one row
    seq  BIGINT NOT NULL,
    hash TEXT   NOT NULL
);

INSERT INTO snapshot_chain_head (id, seq, hash)
VALUES (TRUE, 0, repeat('0', 64));


-- ============================================================
-- Record text (format v1)
--
-- The exact text that is hashed. One field per line, lists
-- sorted byte-wise (COLLATE "C") so the order never depends on
-- the database locale. Measurement values are rounded to 9
-- decimal places; timestamps are UTC with microseconds.
--
--   groundwork-snapshot-v1
--   snapshot_id=<uuid>
--   parcel_id=<uuid>
--   methodology_id=<uuid>
--   methodology_version=<text>
--   date_start=YYYY-MM-DD
--   date_end=YYYY-MM-DD
--   geometry_hash=<as stored>
--   geometry_ewkb_sha256=<hex of the stored geometry's bytes>
--   completed_at=YYYY-MM-DDTHH:MI:SS.USZ
--   measurement=<type slug>:<value>      (one per measurement)
--   scene=<scene id>                     (one per imagery source)
--   prev=<previous record_hash>
--
-- geometry_ewkb_sha256 covers the boundary itself, so editing the
-- geometry is caught even if geometry_hash is left unchanged.
-- ============================================================
CREATE OR REPLACE FUNCTION snapshot_record_text(
    p_snapshot_id  UUID,
    p_completed_at TIMESTAMPTZ,
    p_prev_hash    TEXT
)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-snapshot-v1',
        'snapshot_id=' || s.id,
        'parcel_id=' || s.parcel_id,
        'methodology_id=' || s.methodology_id,
        'methodology_version=' || s.methodology_version,
        'date_start=' || to_char(s.date_start, 'YYYY-MM-DD'),
        'date_end=' || to_char(s.date_end, 'YYYY-MM-DD'),
        'geometry_hash=' || s.geometry_hash,
        'geometry_ewkb_sha256=' || encode(sha256(ST_AsEWKB(s.parcel_geometry, 'NDR')), 'hex'),
        'completed_at=' || to_char(p_completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        (
            SELECT string_agg(
                'measurement=' || mt.slug || ':' || round(m.value::numeric, 9)::text,
                E'\n' ORDER BY mt.slug COLLATE "C"
            )
            FROM measurements m
            JOIN measurement_types mt ON mt.id = m.measurement_type_id
            WHERE m.snapshot_id = s.id
        ),
        (
            SELECT string_agg('scene=' || i.scene_id, E'\n' ORDER BY i.scene_id COLLATE "C")
            FROM imagery_sources i
            WHERE i.snapshot_id = s.id
        ),
        'prev=' || p_prev_hash
    )
    FROM snapshots s
    WHERE s.id = p_snapshot_id;
$$ LANGUAGE sql STABLE;


CREATE OR REPLACE FUNCTION snapshot_record_hash(
    p_snapshot_id  UUID,
    p_completed_at TIMESTAMPTZ,
    p_prev_hash    TEXT
)
RETURNS TEXT AS $$
    SELECT encode(
        sha256(convert_to(snapshot_record_text(p_snapshot_id, p_completed_at, p_prev_hash), 'UTF8')),
        'hex'
    );
$$ LANGUAGE sql STABLE;


-- ============================================================
-- Protection 5: completed snapshots are fully frozen
--
-- Protection 3 only stops the status changing. This blocks any
-- update to a completed row (including its hash columns), stops
-- snapshots being inserted as already completed, and stops the
-- chain columns being set by anyone but the hash trigger.
-- ============================================================
CREATE OR REPLACE FUNCTION freeze_completed_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status = 'completed' THEN
            RAISE EXCEPTION
                'Snapshots cannot be inserted as completed.';
        END IF;
    ELSIF OLD.status = 'completed' THEN
        RAISE EXCEPTION
            'Completed snapshots cannot be changed.';
    END IF;

    IF NEW.status <> 'completed'
       AND (NEW.chain_seq IS NOT NULL
            OR NEW.prev_record_hash IS NOT NULL
            OR NEW.record_hash IS NOT NULL)
    THEN
        RAISE EXCEPTION
            'Hash chain columns are set only when a snapshot completes.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers on the same event fire in name order: freeze (10)
-- runs before the hash chain (20).
CREATE TRIGGER snapshots_10_freeze_completed
BEFORE INSERT OR UPDATE ON snapshots
FOR EACH ROW
EXECUTE FUNCTION freeze_completed_snapshot();


-- ============================================================
-- Hash chain trigger
--
-- Runs once, when a snapshot moves to completed. Measurements and
-- imagery sources must already be inserted (the worker does this
-- in the same transaction, before setting the status).
-- ============================================================
CREATE OR REPLACE FUNCTION stamp_snapshot_hash()
RETURNS TRIGGER AS $$
DECLARE
    head_seq  BIGINT;
    head_hash TEXT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM measurements WHERE snapshot_id = NEW.id) THEN
        RAISE EXCEPTION
            'A snapshot cannot complete without measurements.';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM imagery_sources WHERE snapshot_id = NEW.id) THEN
        RAISE EXCEPTION
            'A snapshot cannot complete without imagery sources.';
    END IF;

    NEW.completed_at := COALESCE(NEW.completed_at, now());

    -- Lock the head; concurrent completions wait here, one at a time.
    SELECT seq, hash
    INTO head_seq, head_hash
    FROM snapshot_chain_head
    WHERE id
    FOR UPDATE;

    NEW.chain_seq        := head_seq + 1;
    NEW.prev_record_hash := head_hash;
    NEW.record_hash      := snapshot_record_hash(NEW.id, NEW.completed_at, head_hash);

    UPDATE snapshot_chain_head
    SET seq = NEW.chain_seq,
        hash = NEW.record_hash
    WHERE id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshots_20_hash_chain
BEFORE UPDATE ON snapshots
FOR EACH ROW
WHEN (OLD.status IS DISTINCT FROM 'completed' AND NEW.status = 'completed')
EXECUTE FUNCTION stamp_snapshot_hash();


-- ============================================================
-- Verification
--
-- Walks the chain in order and returns one row per problem.
-- No rows means the chain is intact.
-- ============================================================
CREATE OR REPLACE FUNCTION verify_snapshot_chain()
RETURNS TABLE (
    bad_seq         BIGINT,
    bad_snapshot_id UUID,
    problem         TEXT
) AS $$
DECLARE
    r             RECORD;
    expected_seq  BIGINT := 0;
    expected_prev TEXT   := repeat('0', 64);
    head_seq      BIGINT;
    head_hash     TEXT;
BEGIN
    FOR r IN
        SELECT s.id, s.chain_seq, s.prev_record_hash, s.record_hash, s.completed_at
        FROM snapshots s
        WHERE s.status = 'completed'
        ORDER BY s.chain_seq NULLS FIRST
    LOOP
        expected_seq := expected_seq + 1;
        bad_seq := r.chain_seq;
        bad_snapshot_id := r.id;

        IF r.chain_seq IS DISTINCT FROM expected_seq THEN
            problem := format('chain_seq is %s, expected %s (missing or extra record)',
                              r.chain_seq, expected_seq);
            RETURN NEXT;
            expected_seq := COALESCE(r.chain_seq, expected_seq);
        END IF;

        IF r.prev_record_hash IS DISTINCT FROM expected_prev THEN
            problem := 'prev_record_hash does not match the previous record_hash';
            RETURN NEXT;
        END IF;

        IF r.record_hash IS DISTINCT FROM
           snapshot_record_hash(r.id, r.completed_at, r.prev_record_hash)
        THEN
            problem := 'record_hash does not match the stored contents';
            RETURN NEXT;
        END IF;

        expected_prev := r.record_hash;
    END LOOP;

    SELECT h.seq, h.hash
    INTO head_seq, head_hash
    FROM snapshot_chain_head h;

    IF head_seq IS DISTINCT FROM expected_seq OR head_hash IS DISTINCT FROM expected_prev THEN
        bad_seq := head_seq;
        bad_snapshot_id := NULL;
        problem := 'chain head does not match the last completed snapshot';
        RETURN NEXT;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE;


-- ============================================================
-- Seed data: first methodology and its measurement types
-- ============================================================
INSERT INTO methodologies (
    name, slug, version, description,
    index_type, satellite, bands, cloud_mask,
    max_cloud_pct, composite_method, date_lookback_days
)
VALUES (
    'NDVI, Sentinel-2 median composite',
    'ndvi-s2-median',
    '1.0.0',
    'Mean and max NDVI over the parcel from a median composite of Sentinel-2 surface reflectance, clouds masked with Cloud Score+.',
    'ndvi', 'sentinel-2', ARRAY['B4', 'B8'], 'cloud_score_plus',
    20, 'median', 30
)
ON CONFLICT (slug, version) DO NOTHING;

INSERT INTO measurement_types (name, slug, unit, description)
VALUES
    ('NDVI mean', 'ndvi_mean', NULL, 'Mean NDVI over the clear pixels of the parcel.'),
    ('NDVI max', 'ndvi_max', NULL, 'Maximum NDVI over the clear pixels of the parcel.'),
    ('Valid pixel percentage', 'valid_pixel_pct', '%', 'Share of the parcel''s pixels that were cloud-free and used.')
ON CONFLICT (slug) DO NOTHING;
