-- ============================================================
-- Groundwork: Initial Database Schema
--
-- Version 1 keeps the schema strict around historical data
-- without adding unnecessary workflow complexity.
-- ============================================================


-- ============================================================
-- Extensions
-- ============================================================

-- Enables PostGIS geometry types and spatial functions (Must be enabled by superuser).
-- CREATE EXTENSION IF NOT EXISTS postgis;

-- Provides gen_random_uuid() (Built-in in Postgres 13+).
-- CREATE EXTENSION IF NOT EXISTS pgcrypto;


-- ============================================================
-- Custom Types
-- ============================================================

-- Snapshot processing states.
CREATE TYPE snapshot_status AS ENUM (
    'pending',
    'processing',
    'completed',
    'failed'
);


-- ============================================================
-- Table: parcels
--
-- Stores parcel boundaries and descriptive information.
-- ============================================================

CREATE TABLE parcels (

    -- Unique parcel identifier.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Parcel name.
    name TEXT NOT NULL,

    -- Optional parcel description.
    description TEXT,

    -- Parcel boundary using WGS 84 coordinates.
    geometry GEOMETRY(POLYGON, 4326) NOT NULL,

    -- Parcel area in hectares.
    area_ha DOUBLE PRECISION
        GENERATED ALWAYS AS (
            ST_Area(geometry::geography) / 10000.0
        ) STORED,

    -- Additional flexible parcel information.
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Last record update time.
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Geometry must not be empty.
    CONSTRAINT parcels_geometry_not_empty_check
        CHECK (NOT ST_IsEmpty(geometry)),

    -- Geometry must be valid.
    CONSTRAINT parcels_geometry_valid_check
        CHECK (ST_IsValid(geometry))
);


-- Spatial index for parcel queries.
CREATE INDEX parcels_geometry_gix
    ON parcels
    USING GIST (geometry);


-- ============================================================
-- Table: methodologies
--
-- Defines how a satellite measurement is calculated.
--
-- Methodologies are immutable. Changes must create a new
-- version instead of modifying an existing record.
-- ============================================================

CREATE TABLE methodologies (

    -- Unique methodology identifier.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Human-readable methodology name.
    name TEXT NOT NULL,

    -- Stable methodology identifier.
    -- Example: ndvi-sentinel2.
    slug TEXT NOT NULL,

    -- Methodology version.
    -- Example: 1.0.0.
    version TEXT NOT NULL,

    -- Optional methodology explanation.
    description TEXT,

    -- Spectral index being calculated.
    -- Examples: ndvi, evi, ndwi.
    index_type TEXT NOT NULL,

    -- Satellite mission used.
    -- Examples: sentinel-2, landsat-9.
    satellite TEXT NOT NULL,

    -- Bands required for the calculation.
    -- Example: ARRAY['B08', 'B04'].
    bands TEXT[] NOT NULL,

    -- Band or dataset used to identify cloudy pixels.
    -- Examples: SCL, QA60.
    cloud_mask TEXT,

    -- Maximum acceptable cloud-cover percentage.
    max_cloud_pct NUMERIC NOT NULL DEFAULT 20,

    -- Method used to combine multiple images.
    composite_method TEXT NOT NULL DEFAULT 'median',

    -- Number of previous days searched for usable imagery.
    date_lookback_days INTEGER NOT NULL DEFAULT 30,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent duplicate versions of the same methodology.
    CONSTRAINT methodologies_slug_version_unique
        UNIQUE (slug, version),

    -- Cloud percentage must be between 0 and 100.
    CONSTRAINT methodologies_cloud_pct_check
        CHECK (max_cloud_pct BETWEEN 0 AND 100),

    -- Lookback period must be positive.
    CONSTRAINT methodologies_lookback_check
        CHECK (date_lookback_days > 0),

    -- At least one band must be provided.
    CONSTRAINT methodologies_bands_check
        CHECK (cardinality(bands) > 0)
);


-- ============================================================
-- Table: snapshots
--
-- Stores historical measurement runs.
--
-- Each snapshot links one parcel, one methodology and one
-- observation period.
-- ============================================================

CREATE TABLE snapshots (

    -- Unique snapshot identifier.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Parcel being analysed.
    parcel_id UUID NOT NULL
        REFERENCES parcels(id),

    -- Methodology used for the analysis.
    methodology_id UUID NOT NULL
        REFERENCES methodologies(id),

    -- Observation period used to select imagery.
    date_start DATE NOT NULL,
    date_end DATE NOT NULL,

    -- Current processing state.
    status snapshot_status NOT NULL DEFAULT 'pending',

    -- Exact parcel geometry used for this run.
    parcel_geometry GEOMETRY(POLYGON, 4326) NOT NULL,

    -- Hash of the parcel geometry used during processing.
    geometry_hash TEXT NOT NULL,

    -- Methodology version used for this run.
    methodology_version TEXT NOT NULL,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Time processing started.
    started_at TIMESTAMPTZ,

    -- Time processing finished.
    completed_at TIMESTAMPTZ,

    -- Optional failure explanation.
    failure_message TEXT,

    -- Observation end cannot precede observation start.
    CONSTRAINT snapshots_date_range_check
        CHECK (date_end >= date_start),

    -- Snapshot geometry must not be empty.
    CONSTRAINT snapshots_geometry_not_empty_check
        CHECK (NOT ST_IsEmpty(parcel_geometry)),

    -- Snapshot geometry must be valid.
    CONSTRAINT snapshots_geometry_valid_check
        CHECK (ST_IsValid(parcel_geometry)),

    -- Completion cannot occur before processing starts.
    CONSTRAINT snapshots_processing_time_check
        CHECK (
            completed_at IS NULL
            OR started_at IS NULL
            OR completed_at >= started_at
        )
);


-- Index snapshots by parcel.
CREATE INDEX snapshots_parcel_id_idx
    ON snapshots (parcel_id);


-- Index snapshots by methodology.
CREATE INDEX snapshots_methodology_id_idx
    ON snapshots (methodology_id);


-- Index snapshots by status.
CREATE INDEX snapshots_status_idx
    ON snapshots (status);


-- Index snapshots by observation period.
CREATE INDEX snapshots_date_range_idx
    ON snapshots (date_start, date_end);


-- ============================================================
-- Table: imagery_sources
--
-- Records the satellite imagery used to create a snapshot.
--
-- A snapshot may use one or more satellite scenes.
-- ============================================================

CREATE TABLE imagery_sources (

    -- Unique imagery record.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Snapshot that used this imagery.
    snapshot_id UUID NOT NULL
        REFERENCES snapshots(id),

    -- Imagery provider.
    provider TEXT NOT NULL,

    -- Satellite mission.
    satellite TEXT NOT NULL,

    -- External scene identifier.
    scene_id TEXT NOT NULL,

    -- Time the image was captured.
    acquisition_time TIMESTAMPTZ NOT NULL,

    -- Scene cloud-cover percentage.
    cloud_cover_pct NUMERIC,

    -- Image processing level.
    processing_level TEXT,

    -- Location or identifier of the imagery asset.
    asset_uri TEXT,

    -- Additional provider-specific information.
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Cloud cover must be between 0 and 100.
    CONSTRAINT imagery_sources_cloud_cover_check
        CHECK (
            cloud_cover_pct IS NULL
            OR cloud_cover_pct BETWEEN 0 AND 100
        ),

    -- Prevent duplicate scenes within one snapshot.
    CONSTRAINT imagery_sources_snapshot_scene_unique
        UNIQUE (snapshot_id, scene_id)
);


-- Index imagery by snapshot.
CREATE INDEX imagery_sources_snapshot_id_idx
    ON imagery_sources (snapshot_id);


-- Index imagery by acquisition time.
CREATE INDEX imagery_sources_acquisition_time_idx
    ON imagery_sources (acquisition_time);


-- ============================================================
-- Table: measurement_types
--
-- Defines metrics Groundwork can calculate.
--
-- New metrics can be added without changing the measurements
-- table.
-- ============================================================

CREATE TABLE measurement_types (

    -- Unique metric definition.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Human-readable metric name.
    name TEXT NOT NULL,

    -- Stable identifier used by the application.
    -- Examples: ndvi_mean, coverage_pct.
    slug TEXT NOT NULL,

    -- Unit used by the metric.
    -- Examples: percent, pixels, hectares.
    unit TEXT,

    -- Short explanation of the metric.
    description TEXT,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent duplicate metric identifiers.
    CONSTRAINT measurement_types_slug_unique
        UNIQUE (slug)
);


-- ============================================================
-- Table: measurements
--
-- Stores calculated metric values for snapshots.
--
-- Each row represents one metric produced by one snapshot.
-- ============================================================

CREATE TABLE measurements (

    -- Unique measurement result.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Snapshot that produced the result.
    snapshot_id UUID NOT NULL
        REFERENCES snapshots(id),

    -- Metric represented by this value.
    measurement_type_id UUID NOT NULL
        REFERENCES measurement_types(id),

    -- Numeric metric value.
    value DOUBLE PRECISION NOT NULL,

    -- Additional measurement-specific information.
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Record creation time.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent duplicate metrics within one snapshot.
    CONSTRAINT measurements_snapshot_type_unique
        UNIQUE (
            snapshot_id,
            measurement_type_id
        )
);


-- Index measurements by snapshot.
CREATE INDEX measurements_snapshot_id_idx
    ON measurements (snapshot_id);


-- Index measurements by metric type.
CREATE INDEX measurements_type_id_idx
    ON measurements (measurement_type_id);


-- ============================================================
-- Trigger: automatically update parcels.updated_at
-- ============================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER parcels_set_updated_at
BEFORE UPDATE ON parcels
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- Protection 1: methodologies are immutable
--
-- Methodologies cannot be updated or deleted.
-- Create a new version instead.
-- ============================================================

CREATE OR REPLACE FUNCTION prevent_methodology_changes()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION
        'Methodologies are immutable. Create a new version instead.';
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER methodologies_prevent_update
BEFORE UPDATE ON methodologies
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();


CREATE TRIGGER methodologies_prevent_delete
BEFORE DELETE ON methodologies
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();


-- ============================================================
-- Protection 2: snapshot inputs are immutable
--
-- Processing fields may change, but the original parcel,
-- methodology, date range and audit fields cannot change.
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
    THEN
        RAISE EXCEPTION
            'Snapshot input and audit fields are immutable.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER snapshots_protect_inputs
BEFORE UPDATE ON snapshots
FOR EACH ROW
EXECUTE FUNCTION protect_snapshot_inputs();


-- ============================================================
-- Protection 3: completed snapshots are final
--
-- Completed snapshots cannot be reopened or deleted.
-- ============================================================

CREATE OR REPLACE FUNCTION protect_completed_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'completed' THEN
            RAISE EXCEPTION
                'Completed snapshots cannot be deleted.';
        END IF;

        RETURN OLD;
    END IF;

    IF OLD.status = 'completed'
       AND NEW.status IS DISTINCT FROM OLD.status
    THEN
        RAISE EXCEPTION
            'Completed snapshots cannot return to another status.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER snapshots_protect_completed_update
BEFORE UPDATE ON snapshots
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot();


CREATE TRIGGER snapshots_protect_completed_delete
BEFORE DELETE ON snapshots
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot();


-- ============================================================
-- Protection 4: completed snapshot children are immutable
--
-- Imagery and measurement records cannot be added, changed or
-- removed after their snapshot is completed.
-- ============================================================

CREATE OR REPLACE FUNCTION protect_completed_snapshot_children()
RETURNS TRIGGER AS $$
DECLARE
    related_snapshot_id UUID;
    related_snapshot_status snapshot_status;
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        SELECT status
        INTO related_snapshot_status
        FROM snapshots
        WHERE id = OLD.snapshot_id;

        IF related_snapshot_status = 'completed' THEN
            RAISE EXCEPTION
                'Records belonging to a completed snapshot cannot be changed.';
        END IF;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        related_snapshot_id := NEW.snapshot_id;

        SELECT status
        INTO related_snapshot_status
        FROM snapshots
        WHERE id = related_snapshot_id;

        IF related_snapshot_status = 'completed' THEN
            RAISE EXCEPTION
                'Records cannot be added to a completed snapshot.';
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER imagery_sources_protect_completed_snapshot
BEFORE INSERT OR UPDATE OR DELETE ON imagery_sources
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot_children();


CREATE TRIGGER measurements_protect_completed_snapshot
BEFORE INSERT OR UPDATE OR DELETE ON measurements
FOR EACH ROW
EXECUTE FUNCTION protect_completed_snapshot_children();