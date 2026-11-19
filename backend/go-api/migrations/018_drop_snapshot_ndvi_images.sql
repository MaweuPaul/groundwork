-- ============================================================
-- Drop snapshot_ndvi_images
--
-- Migration 016 copied its images into artifacts, and nothing has
-- read or written it since. Rows that only recorded a failed
-- drawing (no image) go with it; they were never sealed.
-- ============================================================

-- Refuse to drop it if any image didn't make it into artifacts.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM snapshot_ndvi_images o
        WHERE o.png IS NOT NULL
          AND NOT EXISTS (
              SELECT 1 FROM artifacts a
              WHERE a.snapshot_id = o.snapshot_id AND a.key = 'ndvi_map' AND a.content = o.png
          )
    ) THEN
        RAISE EXCEPTION 'snapshot_ndvi_images has images missing from artifacts; not dropping it.';
    END IF;
END;
$$;

DROP TABLE snapshot_ndvi_images;

DROP FUNCTION check_ndvi_image_snapshot();
