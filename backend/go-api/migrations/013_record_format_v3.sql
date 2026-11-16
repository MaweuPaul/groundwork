-- ============================================================
-- Record format v3: any methodology's parameters, dataset
-- provenance and quality flags are sealed
--
-- Snapshots and series gain a parameters object, filled with the
-- methodology's defaults and checked against methodology_parameters
-- on insert. The two NDVI cloud columns stay for v1/v2 records and
-- as a stand-in for callers that don't send parameters yet.
--
-- v3 seals each source with its version (so reprocessed input
-- data is detectable) and any quality flags the methodology
-- raised. Records sealed before this migration keep their format
-- and verify exactly as before.
-- ============================================================


-- NDVI flags snapshots with less than this share of clear pixels.
INSERT INTO methodology_parameters (methodology_id, name, label, type, min_value, max_value, default_value)
SELECT m.id, 'min_clear_pct', 'Minimum clear pixels (%)', 'number', 0, 100, 50
FROM methodologies m
WHERE m.slug = 'ndvi-s2-median' AND m.version = '1.0.0';


-- ============================================================
-- Parameters: defaults filled in, names and ranges checked
-- ============================================================
CREATE OR REPLACE FUNCTION resolve_methodology_parameters(p_methodology_id UUID, p_given JSONB)
RETURNS JSONB AS $$
DECLARE
    v_spec   RECORD;
    v_name   TEXT;
    v_value  JSONB;
    v_result JSONB := '{}'::jsonb;
BEGIN
    IF p_given IS NOT NULL AND jsonb_typeof(p_given) <> 'object' THEN
        RAISE EXCEPTION 'parameters must be a JSON object.';
    END IF;

    FOR v_name IN SELECT jsonb_object_keys(COALESCE(p_given, '{}'::jsonb)) LOOP
        IF NOT EXISTS (
            SELECT 1 FROM methodology_parameters
            WHERE methodology_id = p_methodology_id AND name = v_name
        ) THEN
            RAISE EXCEPTION 'Unknown parameter "%" for this methodology.', v_name;
        END IF;
    END LOOP;

    FOR v_spec IN
        SELECT * FROM methodology_parameters WHERE methodology_id = p_methodology_id
    LOOP
        v_value := COALESCE(p_given -> v_spec.name, to_jsonb(v_spec.default_value));
        IF jsonb_typeof(v_value) <> 'number' THEN
            RAISE EXCEPTION 'Parameter "%" must be a number.', v_spec.name;
        END IF;
        IF (v_value #>> '{}')::numeric NOT BETWEEN v_spec.min_value AND v_spec.max_value THEN
            RAISE EXCEPTION 'Parameter "%" must be between % and %.',
                v_spec.name, v_spec.min_value, v_spec.max_value;
        END IF;
        v_result := v_result || jsonb_build_object(v_spec.name, v_value);
    END LOOP;

    RETURN v_result;
END;
$$ LANGUAGE plpgsql STABLE;


-- ============================================================
-- Snapshots
-- ============================================================
ALTER TABLE snapshots ADD COLUMN parameters JSONB;

-- Existing rows record the settings they were computed with. Completed
-- rows are frozen by triggers, so they're switched off for this backfill;
-- their record text doesn't read parameters, so their hashes don't change.
ALTER TABLE snapshots DISABLE TRIGGER USER;

UPDATE snapshots
SET parameters = jsonb_build_object(
    'max_scene_cloud_pct', max_scene_cloud_pct,
    'cloud_score_threshold', cloud_score_threshold
);

ALTER TABLE snapshots ENABLE TRIGGER USER;

ALTER TABLE snapshots
    ALTER COLUMN parameters SET NOT NULL,
    ALTER COLUMN max_scene_cloud_pct DROP NOT NULL,
    ALTER COLUMN cloud_score_threshold DROP NOT NULL,
    ALTER COLUMN record_format SET DEFAULT 3,
    DROP CONSTRAINT snapshots_record_format_check,
    ADD CONSTRAINT snapshots_record_format_check CHECK (record_format IN (1, 2, 3)),
    ADD CONSTRAINT snapshots_parameters_object CHECK (jsonb_typeof(parameters) = 'object'),
    ADD CONSTRAINT snapshots_legacy_cloud_columns CHECK (
        record_format >= 3
        OR (max_scene_cloud_pct IS NOT NULL AND cloud_score_threshold IS NOT NULL)
    );


CREATE OR REPLACE FUNCTION set_snapshot_record_format()
RETURNS TRIGGER AS $$
BEGIN
    NEW.record_format := 3;

    -- Callers that still send the two cloud columns instead of parameters.
    IF NEW.parameters IS NULL THEN
        NEW.parameters := jsonb_strip_nulls(jsonb_build_object(
            'max_scene_cloud_pct', NEW.max_scene_cloud_pct,
            'cloud_score_threshold', NEW.cloud_score_threshold
        ));
    END IF;
    NEW.parameters := resolve_methodology_parameters(NEW.methodology_id, NEW.parameters);

    -- Kept in step for readers of the old columns; v3 records seal parameters.
    NEW.max_scene_cloud_pct := COALESCE(NEW.max_scene_cloud_pct, (NEW.parameters ->> 'max_scene_cloud_pct')::numeric);
    NEW.cloud_score_threshold := COALESCE(NEW.cloud_score_threshold, (NEW.parameters ->> 'cloud_score_threshold')::numeric);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


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
       OR NEW.parameters IS DISTINCT FROM OLD.parameters
       OR NEW.record_format IS DISTINCT FROM OLD.record_format
       OR NEW.short_id IS DISTINCT FROM OLD.short_id
       OR NEW.series_id IS DISTINCT FROM OLD.series_id
    THEN
        RAISE EXCEPTION
            'Snapshot input and audit fields are immutable.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Evidence: source versions and quality flags
-- ============================================================

-- The source's version at compute time (Earth Engine's system:version),
-- which changes when the provider reprocesses it.
ALTER TABLE imagery_sources ADD COLUMN source_version TEXT;

CREATE TABLE snapshot_flags (
    snapshot_id UUID NOT NULL REFERENCES snapshots(id),
    flag        TEXT NOT NULL CHECK (flag ~ '^[a-z][a-z0-9_]*$'),
    detail      TEXT,
    PRIMARY KEY (snapshot_id, flag)
);

CREATE TRIGGER snapshot_flags_protect_completed_snapshot
BEFORE INSERT OR UPDATE OR DELETE ON snapshot_flags
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot_children();

GRANT SELECT, INSERT ON snapshot_flags TO gee_worker;


-- ============================================================
-- Record text, format v3
--
--   groundwork-snapshot-v3
--   snapshot_id, parcel_id, methodology_id, methodology_version
--   param=<name>:<value>                 (one per parameter, by name)
--   date_start, date_end, geometry_hash, geometry_ewkb_sha256, completed_at
--   measurement=<type slug>:<value>      (as v1)
--   source=<asset uri>@<source version>  (one per source, replaces scene=)
--   flag=<flag>                          (one per quality flag, if any)
--   prev=<previous record hash>
--
-- Numbers are rounded to 9 decimals, like measurements. The v1 and v2
-- branches are unchanged, so their records produce the same text.
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
        CASE WHEN s.record_format = 2 THEN 'max_scene_cloud_pct=' || s.max_scene_cloud_pct END,
        CASE WHEN s.record_format = 2 THEN 'cloud_score_threshold=' || s.cloud_score_threshold END,
        CASE WHEN s.record_format >= 3 THEN (
            SELECT string_agg(
                'param=' || p.key || ':' || CASE jsonb_typeof(p.value)
                    WHEN 'number' THEN round((p.value #>> '{}')::numeric, 9)::text
                    ELSE p.value #>> '{}'
                END,
                E'\n' ORDER BY p.key COLLATE "C"
            )
            FROM jsonb_each(s.parameters) p
        ) END,
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
        CASE WHEN s.record_format < 3 THEN (
            SELECT string_agg('scene=' || i.scene_id, E'\n' ORDER BY i.scene_id COLLATE "C")
            FROM imagery_sources i
            WHERE i.snapshot_id = s.id
        ) END,
        CASE WHEN s.record_format >= 3 THEN (
            -- A source with no URI falls back to its scene id, so it can never drop out of the record.
            SELECT string_agg(
                'source=' || COALESCE(i.asset_uri, 'scene:' || i.scene_id)
                    || '@' || COALESCE(i.source_version, 'unversioned'),
                E'\n' ORDER BY COALESCE(i.asset_uri, 'scene:' || i.scene_id) COLLATE "C",
                               COALESCE(i.source_version, 'unversioned') COLLATE "C"
            )
            FROM imagery_sources i
            WHERE i.snapshot_id = s.id
        ) END,
        CASE WHEN s.record_format >= 3 THEN (
            SELECT string_agg('flag=' || f.flag, E'\n' ORDER BY f.flag COLLATE "C")
            FROM snapshot_flags f
            WHERE f.snapshot_id = s.id
        ) END,
        'prev=' || p_prev_hash
    )
    FROM snapshots s
    WHERE s.id = p_snapshot_id;
$$ LANGUAGE sql STABLE;


-- ============================================================
-- claim_snapshot_job: returns parameters in place of the two
-- cloud settings. The return type changes, so it's recreated.
-- ============================================================
DROP FUNCTION claim_snapshot_job(TEXT, INTERVAL);

CREATE FUNCTION claim_snapshot_job(
    p_worker   TEXT,
    p_lock_for INTERVAL DEFAULT INTERVAL '15 minutes'
)
RETURNS TABLE (
    job_id              UUID,
    snapshot_id         UUID,
    attempt             INTEGER,
    methodology_id      UUID,
    methodology_version TEXT,
    date_start          DATE,
    date_end            DATE,
    parameters          JSONB,
    geometry_geojson    TEXT
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
           s.parameters,
           ST_AsGeoJSON(s.parcel_geometry)
    FROM snapshots s
    WHERE s.id = v_snapshot_id;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;

REVOKE ALL ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) TO gee_worker;


-- ============================================================
-- Series: parameters too, sealed from series record v2
-- ============================================================
ALTER TABLE snapshot_series
    ADD COLUMN parameters JSONB,
    ADD COLUMN record_format SMALLINT;

-- Sealed series are frozen by a trigger; their v1 text doesn't read these columns.
ALTER TABLE snapshot_series DISABLE TRIGGER USER;

UPDATE snapshot_series
SET parameters = jsonb_build_object(
        'max_scene_cloud_pct', max_scene_cloud_pct,
        'cloud_score_threshold', cloud_score_threshold
    ),
    record_format = 1;

ALTER TABLE snapshot_series ENABLE TRIGGER USER;

ALTER TABLE snapshot_series
    ALTER COLUMN parameters SET NOT NULL,
    ALTER COLUMN record_format SET NOT NULL,
    ALTER COLUMN record_format SET DEFAULT 2,
    ALTER COLUMN max_scene_cloud_pct DROP NOT NULL,
    ALTER COLUMN cloud_score_threshold DROP NOT NULL,
    ADD CONSTRAINT snapshot_series_record_format_check CHECK (record_format IN (1, 2)),
    ADD CONSTRAINT snapshot_series_parameters_object CHECK (jsonb_typeof(parameters) = 'object'),
    ADD CONSTRAINT snapshot_series_legacy_cloud_columns CHECK (
        record_format >= 2
        OR (max_scene_cloud_pct IS NOT NULL AND cloud_score_threshold IS NOT NULL)
    );


CREATE OR REPLACE FUNCTION prepare_snapshot_series()
RETURNS TRIGGER AS $$
BEGIN
    NEW.record_format := 2;

    IF NEW.parameters IS NULL THEN
        NEW.parameters := jsonb_strip_nulls(jsonb_build_object(
            'max_scene_cloud_pct', NEW.max_scene_cloud_pct,
            'cloud_score_threshold', NEW.cloud_score_threshold
        ));
    END IF;
    NEW.parameters := resolve_methodology_parameters(NEW.methodology_id, NEW.parameters);

    NEW.max_scene_cloud_pct := COALESCE(NEW.max_scene_cloud_pct, (NEW.parameters ->> 'max_scene_cloud_pct')::numeric);
    NEW.cloud_score_threshold := COALESCE(NEW.cloud_score_threshold, (NEW.parameters ->> 'cloud_score_threshold')::numeric);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshot_series_prepare
BEFORE INSERT ON snapshot_series
FOR EACH ROW
EXECUTE FUNCTION prepare_snapshot_series();


-- ============================================================
-- Series record text, format v2
--
-- As v1, with param=<name>:<value> lines (by name) in place of
-- the two cloud settings. v1 records produce the same text.
-- ============================================================
CREATE OR REPLACE FUNCTION series_record_text(p_series_id UUID, p_sealed_at TIMESTAMPTZ)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-series-v' || r.record_format,
        'series_id=' || r.id,
        'parcel_id=' || r.parcel_id,
        'methodology_id=' || r.methodology_id,
        'methodology_version=' || r.methodology_version,
        'date_start=' || to_char(r.date_start, 'YYYY-MM-DD'),
        'date_end=' || to_char(r.date_end, 'YYYY-MM-DD'),
        'window_months=' || r.window_months,
        'interval_months=' || r.interval_months,
        CASE WHEN r.record_format = 1 THEN 'max_scene_cloud_pct=' || r.max_scene_cloud_pct END,
        CASE WHEN r.record_format = 1 THEN 'cloud_score_threshold=' || r.cloud_score_threshold END,
        CASE WHEN r.record_format >= 2 THEN (
            SELECT string_agg(
                'param=' || p.key || ':' || CASE jsonb_typeof(p.value)
                    WHEN 'number' THEN round((p.value #>> '{}')::numeric, 9)::text
                    ELSE p.value #>> '{}'
                END,
                E'\n' ORDER BY p.key COLLATE "C"
            )
            FROM jsonb_each(r.parameters) p
        ) END,
        (
            SELECT string_agg(
                'window=' || to_char(s.date_start, 'YYYY-MM-DD') || '/' || to_char(s.date_end, 'YYYY-MM-DD')
                    || ':' || COALESCE(s.record_hash, 'failed'),
                E'\n' ORDER BY s.date_start
            )
            FROM snapshots s
            WHERE s.series_id = r.id
        ),
        'sealed_at=' || to_char(p_sealed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )
    FROM snapshot_series r
    WHERE r.id = p_series_id;
$$ LANGUAGE sql STABLE;
