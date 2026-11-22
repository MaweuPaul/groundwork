-- ============================================================
-- Fourth methodology: NDVI against the parcel's own history
--
-- The window's NDVI next to the mean of the same calendar window in
-- up to five earlier years, so a lender sees whether this season is
-- better or worse than usual for this field. Only earlier years
-- count, so a sealed result doesn't change as new years arrive.
-- Data starts in 2022: three baseline years from 2019, when
-- Sentinel-2 surface reflectance covers the world.
-- ============================================================

INSERT INTO methodologies (
    name, slug, version, description,
    index_type, satellite, bands, cloud_mask,
    composite_method, data_start
)
VALUES (
    'NDVI against earlier years, Sentinel-2',
    'ndvi-anomaly-s2',
    '1.0.0',
    'Mean NDVI over the parcel for the window, made exactly like the NDVI methodology, and the mean of the same calendar window in up to five earlier years from 2019. A baseline year counts only if at least the minimum share of the parcel was clear; at least three must count. The anomaly is the window minus the baseline.',
    'ndvi_anomaly', 'sentinel-2', ARRAY['B4', 'B8'], 'cloud_score_plus',
    'median', '2022-01-01'
);

INSERT INTO measurement_types (name, slug, unit, description)
VALUES
    ('NDVI anomaly',        'ndvi_anomaly',        NULL,    'The window''s mean NDVI minus the baseline; above 0 is greener than usual for this parcel.'),
    ('NDVI baseline',       'ndvi_baseline_mean',  NULL,    'Mean of the parcel''s NDVI for the same calendar window in the counted earlier years.'),
    ('Baseline years',      'ndvi_baseline_years', 'years', 'Earlier years with enough clear pixels to count in the baseline.');

INSERT INTO methodology_measurements (methodology_id, measurement_type_id, position)
SELECT m.id, mt.id, x.position
FROM (VALUES
    ('ndvi_anomaly',        1),
    ('ndvi_mean',           2),
    ('ndvi_baseline_mean',  3),
    ('ndvi_baseline_years', 4),
    ('valid_pixel_pct',     5)
) AS x(measurement, position)
JOIN methodologies m ON m.slug = 'ndvi-anomaly-s2' AND m.version = '1.0.0'
JOIN measurement_types mt ON mt.slug = x.measurement;

INSERT INTO methodology_parameters (methodology_id, name, label, type, min_value, max_value, default_value)
SELECT m.id, x.name, x.label, 'number', x.min_value, x.max_value, x.default_value
FROM (VALUES
    ('max_scene_cloud_pct',   'Max scene cloud cover (%)', 0.01, 100,  20),
    ('cloud_score_threshold', 'Clear pixel threshold',     0.40, 0.80, 0.60),
    ('min_clear_pct',         'Minimum clear pixels (%)',  0,    100,  50)
) AS x(name, label, min_value, max_value, default_value)
JOIN methodologies m ON m.slug = 'ndvi-anomaly-s2' AND m.version = '1.0.0';
