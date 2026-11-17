-- ============================================================
-- Artifacts, sealed into records (record format v5)
--
-- An artifact is a file a methodology produces with its result,
-- e.g. the NDVI map image. The worker stores it in the same
-- transaction as the measurements, before the snapshot completes,
-- so v5 records seal one line per artifact, sorted by key:
--
--   artifact=<key>:<media type>:<sha256 of content>:<sha256 of metadata>
--
-- The database computes both hashes itself, so they can't disagree
-- with what is stored. Metadata (e.g. where to draw an image) is
-- hashed as jsonb text, which Postgres writes in one fixed form.
--
-- snapshot_ndvi_images is no longer written. Its images are copied
-- here, but the records of those snapshots predate v5 and don't
-- cover them. The old table is dropped in a later migration, once
-- no running service reads it.
-- ============================================================

-- SHA-256 of jsonb's text form. Postgres writes jsonb in one fixed form (keys sorted,
-- no insignificant whitespace) and the encoding is pinned to UTF-8, so the result depends
-- only on the value; convert_to is merely marked stable, hence the IMMUTABLE wrapper.
CREATE FUNCTION jsonb_sha256(p_value JSONB)
RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(p_value::text, 'UTF8')), 'hex');
$$ LANGUAGE sql IMMUTABLE;

CREATE TABLE artifacts (
    snapshot_id     UUID  NOT NULL REFERENCES snapshots(id),
    key             TEXT  NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    media_type      TEXT  NOT NULL CHECK (media_type ~ '^[a-z]+/[a-z0-9.+-]+$'),
    content         BYTEA NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata) = 'object'),
    sha256          TEXT  GENERATED ALWAYS AS (encode(sha256(content), 'hex')) STORED,
    metadata_sha256 TEXT  GENERATED ALWAYS AS (jsonb_sha256(metadata)) STORED,
    byte_size       INTEGER GENERATED ALWAYS AS (octet_length(content)) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (snapshot_id, key)
);

INSERT INTO artifacts (snapshot_id, key, media_type, content, metadata, created_at)
SELECT snapshot_id, 'ndvi_map', 'image/png', png,
       jsonb_build_object(
           'width', width, 'height', height,
           'bounds', jsonb_build_array(west, south, east, north),
           'visualization', visualization
       ),
       created_at
FROM snapshot_ndvi_images
WHERE png IS NOT NULL;

-- Like measurements: nothing is added to, changed in or removed from a completed snapshot.
CREATE TRIGGER artifacts_protect_completed_snapshot
BEFORE INSERT OR UPDATE OR DELETE ON artifacts
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot_children();

GRANT SELECT, INSERT ON artifacts TO gee_worker;

ALTER TABLE snapshots
    ALTER COLUMN record_format SET DEFAULT 5,
    DROP CONSTRAINT snapshots_record_format_check,
    ADD CONSTRAINT snapshots_record_format_check CHECK (record_format IN (1, 2, 3, 4, 5));


CREATE OR REPLACE FUNCTION set_snapshot_record_format()
RETURNS TRIGGER AS $$
BEGIN
    NEW.record_format := 5;

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
-- Record text, format v5: v4 plus one line per artifact
-- (v1 to v4 branches unchanged)
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
        CASE WHEN s.record_format >= 5 THEN (
            SELECT string_agg(
                'artifact=' || a.key || ':' || a.media_type || ':' || a.sha256 || ':' || a.metadata_sha256,
                E'
' ORDER BY a.key COLLATE "C"
            )
            FROM artifacts a
            WHERE a.snapshot_id = s.id
        ) END,
        'prev=' || p_prev_hash
    )
    FROM snapshots s
    WHERE s.id = p_snapshot_id;
$$ LANGUAGE sql STABLE;
