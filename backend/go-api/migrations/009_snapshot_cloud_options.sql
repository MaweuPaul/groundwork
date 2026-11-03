-- ============================================================
-- Per-snapshot cloud options, sealed into the record
--
-- The methodology's max_cloud_pct is now only the default. Each
-- snapshot stores the two cloud settings it was computed with:
--
--   max_scene_cloud_pct    drop whole Sentinel-2 scenes cloudier
--                          than this (scene-level prefilter)
--   cloud_score_threshold  Cloud Score+ cs_cdf a pixel needs to
--                          count as clear (pixel-level mask)
--
-- Both are snapshot inputs: immutable once created, and part of
-- the hashed record text from format v2 on. Records sealed before
-- this migration stay format v1, so their hashes still verify.
-- ============================================================

ALTER TABLE snapshots
    ADD COLUMN max_scene_cloud_pct   NUMERIC(5, 2),
    ADD COLUMN cloud_score_threshold NUMERIC(4, 3),
    ADD COLUMN record_format         SMALLINT;

-- Existing rows get the values they were actually computed with:
-- the methodology's limit and the fixed 0.60 threshold. Completed rows
-- are frozen by triggers, so they're switched off for this one backfill.
ALTER TABLE snapshots DISABLE TRIGGER USER;

UPDATE snapshots s
SET max_scene_cloud_pct   = m.max_cloud_pct,
    cloud_score_threshold = 0.600,
    record_format         = CASE WHEN s.status = 'completed' THEN 1 ELSE 2 END
FROM methodologies m
WHERE m.id = s.methodology_id;

ALTER TABLE snapshots ENABLE TRIGGER USER;

ALTER TABLE snapshots
    ALTER COLUMN max_scene_cloud_pct   SET NOT NULL,
    ALTER COLUMN cloud_score_threshold SET NOT NULL,
    ALTER COLUMN record_format         SET NOT NULL,
    ALTER COLUMN record_format         SET DEFAULT 2,
    ADD CONSTRAINT snapshots_max_scene_cloud_pct_check
        CHECK (max_scene_cloud_pct > 0 AND max_scene_cloud_pct <= 100),
    ADD CONSTRAINT snapshots_cloud_score_threshold_check
        CHECK (cloud_score_threshold > 0 AND cloud_score_threshold < 1),
    ADD CONSTRAINT snapshots_record_format_check
        CHECK (record_format IN (1, 2));


-- ============================================================
-- New snapshots always use the current record format
--
-- Format v1 leaves the cloud settings out of the hash; only
-- records sealed before this migration may use it.
-- ============================================================
CREATE OR REPLACE FUNCTION set_snapshot_record_format()
RETURNS TRIGGER AS $$
BEGIN
    NEW.record_format := 2;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshots_06_record_format
BEFORE INSERT ON snapshots
FOR EACH ROW
EXECUTE FUNCTION set_snapshot_record_format();


-- ============================================================
-- Protection 2, extended: the new columns are inputs too
-- ============================================================
CREATE OR REPLACE FUNCTION protect_snapshot_inputs()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parcel_id IS DISTINCT FROM OLD.parcel_id
       OR NEW.methodology_id IS DISTINCT FROM OLD.methodology_id
       OR NEW.date_start IS DISTINCT FROM OLD.date_start
       OR NEW.date_end IS DISTINCT FROM OLD.date_end
       OR NEW.parcel_geometry IS DISTINCT FROM OLD.parcel_geometry
       OR NEW.geometry_hash IS DISTINCT FROM OLD.geometry_hash
       OR NEW.methodology_version IS DISTINCT FROM OLD.methodology_version
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.max_scene_cloud_pct IS DISTINCT FROM OLD.max_scene_cloud_pct
       OR NEW.cloud_score_threshold IS DISTINCT FROM OLD.cloud_score_threshold
       OR NEW.record_format IS DISTINCT FROM OLD.record_format
       OR NEW.short_id IS DISTINCT FROM OLD.short_id
    THEN
        RAISE EXCEPTION
            'Snapshot input and audit fields are immutable.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Record text, format v2
--
-- Format v1 plus the two cloud settings, right after the
-- methodology. Values are written as stored, e.g. 60.00 and 0.600.
--
--   groundwork-snapshot-v2
--   ...                                  (as v1 up to methodology_version)
--   max_scene_cloud_pct=<numeric(5,2)>
--   cloud_score_threshold=<numeric(4,3)>
--   ...                                  (as v1 from date_start on)
--
-- concat_ws skips NULLs, so a v1 record's text is unchanged byte for byte.
-- ============================================================
CREATE OR REPLACE FUNCTION snapshot_record_text(
    p_snapshot_id  UUID,
    p_completed_at TIMESTAMPTZ,
    p_prev_hash    TEXT
)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-snapshot-v' || s.record_format,
        'snapshot_id=' || s.id,
        'parcel_id=' || s.parcel_id,
        'methodology_id=' || s.methodology_id,
        'methodology_version=' || s.methodology_version,
        CASE WHEN s.record_format >= 2 THEN 'max_scene_cloud_pct=' || s.max_scene_cloud_pct END,
        CASE WHEN s.record_format >= 2 THEN 'cloud_score_threshold=' || s.cloud_score_threshold END,
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


-- ============================================================
-- claim_snapshot_job: now also returns the cloud settings
--
-- The return type changes, so the function is dropped and
-- recreated, with its grants.
-- ============================================================
DROP FUNCTION claim_snapshot_job(TEXT, INTERVAL);

CREATE FUNCTION claim_snapshot_job(
    p_worker   TEXT,
    p_lock_for INTERVAL DEFAULT INTERVAL '15 minutes'
)
RETURNS TABLE (
    job_id                UUID,
    snapshot_id           UUID,
    attempt               INTEGER,
    methodology_id        UUID,
    methodology_version   TEXT,
    date_start            DATE,
    date_end              DATE,
    max_scene_cloud_pct   NUMERIC,
    cloud_score_threshold NUMERIC,
    geometry_geojson      TEXT
) AS $$
#variable_conflict use_column
DECLARE
    v_job_id      UUID;
    v_snapshot_id UUID;
    v_attempt     INTEGER;
BEGIN
    IF p_worker IS NULL OR btrim(p_worker) = '' THEN
        RAISE EXCEPTION 'A worker id is required.';
    END IF;

    -- Abandoned jobs with no attempts left: fail them for good.
    WITH dead AS (
        UPDATE snapshot_jobs j
        SET status          = 'failed',
            locked_by       = NULL,
            locked_until    = NULL,
            last_error_code = 'LOCK_EXPIRED',
            last_error      = 'Worker stopped responding and no attempts are left.',
            updated_at      = NOW()
        WHERE j.status = 'running'
          AND j.locked_until < NOW()
          AND j.attempts >= j.max_attempts
        RETURNING j.snapshot_id
    )
    UPDATE snapshots s
    SET status = 'failed',
        failure_message = 'LOCK_EXPIRED: worker stopped responding and no attempts are left.'
    WHERE s.id IN (SELECT dead.snapshot_id FROM dead);

    -- Next job: queued and due, or abandoned by a dead worker.
    -- SKIP LOCKED lets several workers claim at once without
    -- waiting on, or taking, the same job.
    UPDATE snapshot_jobs j
    SET status       = 'running',
        attempts     = j.attempts + 1,
        locked_by    = p_worker,
        locked_until = NOW() + p_lock_for,
        updated_at   = NOW()
    WHERE j.id = (
        SELECT c.id
        FROM snapshot_jobs c
        WHERE (c.status = 'queued' AND c.run_after <= NOW())
           OR (c.status = 'running' AND c.locked_until < NOW())
        ORDER BY c.run_after, c.created_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING j.id, j.snapshot_id, j.attempts
    INTO v_job_id, v_snapshot_id, v_attempt;

    IF v_job_id IS NULL THEN
        RETURN;
    END IF;

    -- Keep the first start time when a job is retried.
    UPDATE snapshots s
    SET status = 'processing',
        started_at = COALESCE(s.started_at, NOW())
    WHERE s.id = v_snapshot_id;

    RETURN QUERY
    SELECT v_job_id,
           s.id,
           v_attempt,
           s.methodology_id,
           s.methodology_version,
           s.date_start,
           s.date_end,
           s.max_scene_cloud_pct,
           s.cloud_score_threshold,
           ST_AsGeoJSON(s.parcel_geometry)
    FROM snapshots s
    WHERE s.id = v_snapshot_id;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;

REVOKE ALL ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) TO gee_worker;
