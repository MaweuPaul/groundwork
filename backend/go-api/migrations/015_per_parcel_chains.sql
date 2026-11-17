-- ============================================================
-- One hash chain per parcel
--
-- Until now every completed snapshot linked to the one completed
-- just before it, whatever its parcel, through one global head.
-- From here each parcel has its own chain: a record links only to
-- the previous record of the same parcel, so a parcel's history
-- can be verified on its own, without seeing anyone else's
-- records, and parcels no longer wait on each other to complete.
--
-- Records already sealed stay exactly as they are and keep
-- verifying against the closed global chain. Each parcel's chain
-- continues from that parcel's latest such record, or starts at
-- the genesis hash (64 zeros) when it has none.
--
-- The record text is unchanged: the link is still prev=<hash>,
-- which now names the same parcel's previous record.
-- ============================================================

ALTER TABLE snapshots
    ADD COLUMN parcel_seq BIGINT,
    ADD CONSTRAINT snapshots_parcel_seq_unique UNIQUE (parcel_id, parcel_seq);


-- ============================================================
-- Parcel chain heads
--
-- One row per parcel with its latest position and hash. Completing
-- a snapshot locks its parcel's row only, so two snapshots of the
-- same parcel can't link to the same previous hash.
-- ============================================================
CREATE TABLE parcel_chain_heads (
    parcel_id UUID PRIMARY KEY REFERENCES parcels(id),
    seq       BIGINT NOT NULL,
    hash      TEXT   NOT NULL
);

-- Start each parcel that already has sealed records from its latest one.
INSERT INTO parcel_chain_heads (parcel_id, seq, hash)
SELECT DISTINCT ON (parcel_id)
       parcel_id,
       count(*) OVER (PARTITION BY parcel_id),
       record_hash
FROM snapshots
WHERE status = 'completed'
ORDER BY parcel_id, chain_seq DESC;


-- The record's position in its parcel's chain: stored for records sealed per parcel,
-- counted for earlier ones. Null until the snapshot completes.
CREATE FUNCTION parcel_record_number(s snapshots)
RETURNS BIGINT AS $$
    SELECT CASE
        WHEN s.status <> 'completed' THEN NULL
        WHEN s.parcel_seq IS NOT NULL THEN s.parcel_seq
        ELSE (
            SELECT count(*)
            FROM snapshots x
            WHERE x.parcel_id = s.parcel_id
              AND x.status = 'completed'
              AND x.parcel_seq IS NULL
              AND x.chain_seq <= s.chain_seq
        )
    END;
$$ LANGUAGE sql STABLE;


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
            OR NEW.parcel_seq IS NOT NULL
            OR NEW.prev_record_hash IS NOT NULL
            OR NEW.record_hash IS NOT NULL)
    THEN
        RAISE EXCEPTION
            'Hash chain columns are set only when a snapshot completes.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Hash chain trigger: links to the parcel's previous record
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

    -- A parcel's first record starts its chain from the genesis hash.
    INSERT INTO parcel_chain_heads (parcel_id, seq, hash)
    VALUES (NEW.parcel_id, 0, repeat('0', 64))
    ON CONFLICT (parcel_id) DO NOTHING;

    -- Lock this parcel's head; completions of the same parcel wait here, one at a time.
    SELECT seq, hash
    INTO head_seq, head_hash
    FROM parcel_chain_heads
    WHERE parcel_id = NEW.parcel_id
    FOR UPDATE;

    NEW.parcel_seq       := head_seq + 1;
    NEW.prev_record_hash := head_hash;
    NEW.record_hash      := snapshot_record_hash(NEW.id, NEW.completed_at, head_hash);

    UPDATE parcel_chain_heads
    SET seq = NEW.parcel_seq,
        hash = NEW.record_hash
    WHERE parcel_id = NEW.parcel_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Verification
--
-- verify_parcel_chain walks one parcel's records in order: the
-- ones sealed in the global chain have their contents rechecked
-- (their links point across parcels and are checked by
-- verify_snapshot_chain), then each per-parcel record must follow
-- the one before it. No rows means the parcel's chain is intact.
-- ============================================================
CREATE FUNCTION verify_parcel_chain(p_parcel_id UUID)
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
        SELECT s.id, s.chain_seq, s.parcel_seq, s.prev_record_hash, s.record_hash, s.completed_at
        FROM snapshots s
        WHERE s.parcel_id = p_parcel_id AND s.status = 'completed'
        ORDER BY s.parcel_seq NULLS FIRST, s.chain_seq
    LOOP
        expected_seq := expected_seq + 1;
        bad_seq := COALESCE(r.parcel_seq, expected_seq);
        bad_snapshot_id := r.id;

        IF r.parcel_seq IS NOT NULL THEN
            IF r.parcel_seq IS DISTINCT FROM expected_seq THEN
                problem := format('parcel_seq is %s, expected %s (missing or extra record)',
                                  r.parcel_seq, expected_seq);
                RETURN NEXT;
                expected_seq := r.parcel_seq;
            END IF;

            IF r.prev_record_hash IS DISTINCT FROM expected_prev THEN
                problem := 'prev_record_hash does not match the parcel''s previous record_hash';
                RETURN NEXT;
            END IF;
        END IF;

        IF r.record_hash IS DISTINCT FROM
           snapshot_record_hash(r.id, r.completed_at, r.prev_record_hash)
        THEN
            problem := 'record_hash does not match the stored contents';
            RETURN NEXT;
        END IF;

        expected_prev := r.record_hash;
    END LOOP;

    -- The head remembers the chain's length, so records removed from its end still show.
    SELECT h.seq, h.hash
    INTO head_seq, head_hash
    FROM parcel_chain_heads h
    WHERE h.parcel_id = p_parcel_id;

    IF (FOUND OR expected_seq > 0)
       AND (head_seq IS DISTINCT FROM expected_seq OR head_hash IS DISTINCT FROM expected_prev)
    THEN
        bad_seq := head_seq;
        bad_snapshot_id := NULL;
        problem := 'parcel chain head does not match the parcel''s last completed snapshot';
        RETURN NEXT;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE;


-- ============================================================
-- verify_snapshot_chain: the closed global chain, then every
-- parcel's chain. No rows means everything is intact.
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
    -- Records sealed before per-parcel chains, in their global order.
    FOR r IN
        SELECT s.id, s.chain_seq, s.prev_record_hash, s.record_hash
        FROM snapshots s
        WHERE s.status = 'completed' AND s.parcel_seq IS NULL
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

        expected_prev := r.record_hash;
    END LOOP;

    SELECT h.seq, h.hash
    INTO head_seq, head_hash
    FROM snapshot_chain_head h;

    IF head_seq IS DISTINCT FROM expected_seq OR head_hash IS DISTINCT FROM expected_prev THEN
        bad_seq := head_seq;
        bad_snapshot_id := NULL;
        problem := 'global chain head does not match the last globally chained snapshot';
        RETURN NEXT;
    END IF;

    -- Every parcel's chain; this also rechecks every record's contents.
    RETURN QUERY
    SELECT v.bad_seq, v.bad_snapshot_id, v.problem
    FROM (
        SELECT DISTINCT parcel_id FROM snapshots WHERE status = 'completed'
        UNION
        SELECT parcel_id FROM parcel_chain_heads
    ) p
    CROSS JOIN LATERAL verify_parcel_chain(p.parcel_id) v;
END;
$$ LANGUAGE plpgsql STABLE;
