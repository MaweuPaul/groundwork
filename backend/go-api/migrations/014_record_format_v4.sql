-- ============================================================
-- Record format v4: seal what produced each result
--
-- v4 records add two lines after methodology_version:
--   manifest_sha256=<hash of the methodology's published manifest>
--   code_version=<git commit of the worker that computed it>
--
-- The manifest hash is computed from methodologies and
-- methodology_parameters, which are immutable and which the worker
-- checks its manifest against at startup, so it is never stored.
-- v1 to v3 records keep producing the same text and still verify.
-- ============================================================

ALTER TABLE snapshots
    ADD COLUMN code_version TEXT;

ALTER TABLE snapshots
    ALTER COLUMN record_format SET DEFAULT 4,
    DROP CONSTRAINT snapshots_record_format_check,
    ADD CONSTRAINT snapshots_record_format_check CHECK (record_format IN (1, 2, 3, 4)),
    ADD CONSTRAINT snapshots_code_version_sealed CHECK (
        record_format < 4 OR status <> 'completed' OR code_version IS NOT NULL
    );


-- ============================================================
-- Manifest text and hash
--
--   groundwork-manifest-v1
--   slug, version
--   param=<name>:<type>:<min>:<max>:<default>   (by name; "-" when unset)
--
-- Labels are display text, not part of the recipe, so they're left out.
-- ============================================================
CREATE FUNCTION methodology_manifest_text(p_methodology_id UUID)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-manifest-v1',
        'slug=' || m.slug,
        'version=' || m.version,
        (
            SELECT string_agg(
                'param=' || p.name || ':' || p.type
                    || ':' || COALESCE(round(p.min_value::numeric, 9)::text, '-')
                    || ':' || COALESCE(round(p.max_value::numeric, 9)::text, '-')
                    || ':' || COALESCE(round(p.default_value::numeric, 9)::text, '-'),
                E'\n' ORDER BY p.name COLLATE "C"
            )
            FROM methodology_parameters p
            WHERE p.methodology_id = m.id
        )
    )
    FROM methodologies m
    WHERE m.id = p_methodology_id;
$$ LANGUAGE sql STABLE;

CREATE FUNCTION methodology_manifest_hash(p_methodology_id UUID)
RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(methodology_manifest_text(p_methodology_id), 'UTF8')), 'hex');
$$ LANGUAGE sql STABLE;


CREATE OR REPLACE FUNCTION set_snapshot_record_format()
RETURNS TRIGGER AS $$
BEGIN
    NEW.record_format := 4;

    -- Callers that still send the two cloud columns instead of parameters.
    IF NEW.parameters IS NULL THEN
        NEW.parameters := jsonb_strip_nulls(jsonb_build_object(
            'max_scene_cloud_pct', NEW.max_scene_cloud_pct,
            'cloud_score_threshold', NEW.cloud_score_threshold
        ));
    END IF;
    NEW.parameters := resolve_methodology_parameters(NEW.methodology_id, NEW.parameters);

    -- Kept in step for readers of the old columns; v3+ records seal parameters.
    NEW.max_scene_cloud_pct := COALESCE(NEW.max_scene_cloud_pct, (NEW.parameters ->> 'max_scene_cloud_pct')::numeric);
    NEW.cloud_score_threshold := COALESCE(NEW.cloud_score_threshold, (NEW.parameters ->> 'cloud_score_threshold')::numeric);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Record text, format v4 (v1 to v3 branches unchanged)
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
        CASE WHEN s.record_format >= 4 THEN 'manifest_sha256=' || methodology_manifest_hash(s.methodology_id) END,
        CASE WHEN s.record_format >= 4 THEN 'code_version=' || s.code_version END,
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
-- complete_snapshot_job(job, worker, code_version)
--
-- The code version is written in its own UPDATE first: the hash
-- chain trigger runs before the status update and reads the
-- snapshot from the table, so it must already hold the version.
-- The default keeps the two-argument call valid for a worker still
-- running the old code during a deploy: it can complete v1 to v3
-- snapshots, and its v4 ones are refused rather than sealed
-- without a version.
-- ============================================================
DROP FUNCTION complete_snapshot_job(UUID, TEXT);

CREATE FUNCTION complete_snapshot_job(
    p_job_id       UUID,
    p_worker       TEXT,
    p_code_version TEXT DEFAULT NULL
)
RETURNS TEXT AS $$
DECLARE
    job  snapshot_jobs;
    hash TEXT;
BEGIN
    job := lock_owned_snapshot_job(p_job_id, p_worker);

    UPDATE snapshots
    SET code_version = NULLIF(btrim(p_code_version), '')
    WHERE id = job.snapshot_id;

    UPDATE snapshots
    SET status = 'completed'
    WHERE id = job.snapshot_id
    RETURNING record_hash INTO hash;

    UPDATE snapshot_jobs
    SET status          = 'succeeded',
        locked_by       = NULL,
        locked_until    = NULL,
        last_error_code = NULL,
        last_error      = NULL,
        updated_at      = NOW()
    WHERE id = p_job_id;

    RETURN hash;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;

REVOKE ALL ON FUNCTION complete_snapshot_job(UUID, TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION complete_snapshot_job(UUID, TEXT, TEXT) TO gee_worker;
