-- ============================================================
-- Parcels: ownership and multi-part boundaries
-- ============================================================

-- Owner of the parcel. RESTRICT: a user with parcels can't be deleted,
-- because snapshots must keep pointing at a real parcel.
ALTER TABLE parcels
    ADD COLUMN owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT;

CREATE INDEX parcels_owner_id_idx
    ON parcels (owner_id);


-- One parcel can have several pieces (split by a river, road, etc.).
-- Every parcel is stored as a MultiPolygon; a single piece is a MultiPolygon with one part.

-- area_ha is generated from geometry, so Postgres won't change geometry's type
-- while it exists. Drop it, change the type, add it back unchanged.
ALTER TABLE parcels DROP COLUMN area_ha;

ALTER TABLE parcels
    ALTER COLUMN geometry TYPE GEOMETRY(MULTIPOLYGON, 4326)
    USING ST_Multi(geometry);

ALTER TABLE parcels
    ADD COLUMN area_ha DOUBLE PRECISION
        GENERATED ALWAYS AS (
            ST_Area(geometry::geography) / 10000.0
        ) STORED;


-- Snapshots freeze the parcel's geometry, so they need the same type.
ALTER TABLE snapshots
    ALTER COLUMN parcel_geometry TYPE GEOMETRY(MULTIPOLYGON, 4326)
    USING ST_Multi(parcel_geometry);
