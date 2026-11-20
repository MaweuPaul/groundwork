-- ============================================================
-- Methodology measurements
--
-- Which measurements each methodology produces, in display order,
-- so the API can publish them with the methodology. They mirror
-- the plugin manifest's `measurements`. Insert-only, like
-- parameters: a different set means a new methodology version.
-- ============================================================
CREATE TABLE methodology_measurements (
    methodology_id      UUID     NOT NULL REFERENCES methodologies(id),
    measurement_type_id UUID     NOT NULL REFERENCES measurement_types(id),
    position            SMALLINT NOT NULL,
    PRIMARY KEY (methodology_id, measurement_type_id),
    UNIQUE (methodology_id, position)
);

CREATE TRIGGER methodology_measurements_prevent_update
BEFORE UPDATE ON methodology_measurements
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();

CREATE TRIGGER methodology_measurements_prevent_delete
BEFORE DELETE ON methodology_measurements
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();

INSERT INTO methodology_measurements (methodology_id, measurement_type_id, position)
SELECT m.id, mt.id, x.position
FROM (VALUES
    ('ndvi-s2-median',     'ndvi_mean',              1),
    ('ndvi-s2-median',     'ndvi_max',               2),
    ('ndvi-s2-median',     'valid_pixel_pct',        3),
    ('chirps-rain-season', 'rain_total_mm',          1),
    ('chirps-rain-season', 'rain_normal_mm',         2),
    ('chirps-rain-season', 'rain_pct_of_normal',     3),
    ('chirps-rain-season', 'rainy_days',             4),
    ('chirps-rain-season', 'longest_dry_spell_days', 5)
) AS x(slug, measurement, position)
JOIN methodologies m ON m.slug = x.slug AND m.version = '1.0.0'
JOIN measurement_types mt ON mt.slug = x.measurement;

GRANT SELECT ON methodology_measurements TO gee_worker;
