-- ============================================================
-- Second methodology: seasonal rainfall from CHIRPS
--
-- Adds data_start so each methodology states when its data
-- begins; the API checks snapshot dates against it. The new
-- column fills the existing NDVI row with Sentinel-2's first day
-- without firing the methodology update triggers.
-- ============================================================

ALTER TABLE methodologies
    ADD COLUMN data_start DATE NOT NULL DEFAULT '2015-06-23';

ALTER TABLE methodologies
    ALTER COLUMN data_start DROP DEFAULT;

INSERT INTO methodologies (
    name, slug, version, description,
    index_type, satellite, bands, cloud_mask,
    composite_method, data_start
)
VALUES (
    'Seasonal rainfall, CHIRPS',
    'chirps-rain-season',
    '1.0.0',
    'Rainfall over the parcel from CHIRPS daily data: the window''s total, the 1991-2020 average for the same calendar window, rainy days and the longest dry spell. A CHIRPS pixel is about 5.5 km across, so values describe the area around the parcel.',
    'rainfall', 'chirps', ARRAY['precipitation'], NULL,
    'sum', '1981-01-01'
);

INSERT INTO measurement_types (name, slug, unit, description)
VALUES
    ('Rainfall total',        'rain_total_mm',          'mm',   'Total CHIRPS rainfall over the window, averaged over the parcel.'),
    ('Normal rainfall',       'rain_normal_mm',         'mm',   'Mean of the 1991-2020 totals for the same calendar window.'),
    ('Rainfall, % of normal', 'rain_pct_of_normal',     '%',    'Rainfall total as a percentage of the normal; below 100 is drier than usual.'),
    ('Rainy days',            'rainy_days',             'days', 'Days in the window with at least the rainy day threshold.'),
    ('Longest dry spell',     'longest_dry_spell_days', 'days', 'Longest run of consecutive days in the window below the rainy day threshold.');

INSERT INTO methodology_parameters (methodology_id, name, label, type, min_value, max_value, default_value)
SELECT m.id, 'rainy_day_mm', 'Rainy day threshold (mm)', 'number', 0.1, 20, 1
FROM methodologies m
WHERE m.slug = 'chirps-rain-season' AND m.version = '1.0.0';
