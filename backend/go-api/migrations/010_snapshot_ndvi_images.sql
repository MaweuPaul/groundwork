-- ============================================================
-- NDVI images for the map
--
-- After a snapshot completes, the worker renders the same NDVI
-- composite it measured (same scenes, dates and cloud settings,
-- clipped to the frozen boundary) as a PNG in Web Mercator, so the
-- frontend can lay it over the map between its bounds.
--
-- The image is a picture of the evidence, not evidence: it isn't
-- part of the hashed record, so it can be added after the snapshot
-- is sealed. One row per snapshot; a failed render stores the error
-- instead of the image and isn't retried automatically.
-- ============================================================
CREATE TABLE snapshot_ndvi_images (
    snapshot_id UUID PRIMARY KEY
        REFERENCES snapshots(id),

    png    BYTEA,
    width  INTEGER,
    height INTEGER,

    -- Bounding box of the image, WGS84 degrees.
    west  DOUBLE PRECISION,
    south DOUBLE PRECISION,
    east  DOUBLE PRECISION,
    north DOUBLE PRECISION,

    -- How NDVI values were coloured: {"min": .., "max": .., "palette": [..]}.
    visualization JSONB,

    error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT snapshot_ndvi_images_result_check
        CHECK (
            (png IS NOT NULL AND error IS NULL
             AND width > 0 AND height > 0
             AND west < east AND south < north
             AND visualization IS NOT NULL)
            OR (png IS NULL AND error IS NOT NULL)
        )
);

-- Only completed snapshots get an image.
CREATE OR REPLACE FUNCTION check_ndvi_image_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM snapshots WHERE id = NEW.snapshot_id AND status = 'completed') THEN
        RAISE EXCEPTION 'NDVI images are only stored for completed snapshots.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshot_ndvi_images_completed_only
BEFORE INSERT ON snapshot_ndvi_images
FOR EACH ROW
EXECUTE FUNCTION check_ndvi_image_snapshot();

GRANT SELECT, INSERT ON snapshot_ndvi_images TO gee_worker;
