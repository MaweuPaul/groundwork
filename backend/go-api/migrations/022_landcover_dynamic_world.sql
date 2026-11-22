-- ============================================================
-- Third methodology: land cover from Dynamic World
--
-- What share of the parcel is crops, trees, grass and shrub,
-- buildings, water or bare ground in a window, so a lender can see
-- whether the land is farmed.
-- ============================================================

INSERT INTO methodologies (
    name, slug, version, description,
    index_type, satellite, bands, cloud_mask,
    composite_method, data_start
)
VALUES (
    'Land cover, Dynamic World',
    'landcover-dynamic-world',
    '1.0.0',
    'Share of the parcel in each land cover class from Google and WRI''s Dynamic World, 10 m classes made from every Sentinel-2 view with at most 35% cloud. Each pixel takes its most common class in the window. Grass and shrub are one share, flooded vegetation counts as water, and snow as bare ground. Shares are of the classified pixels, so they add up to 100.',
    'landcover', 'sentinel-2', ARRAY['label'], NULL,
    'mode', '2015-06-27'
);

INSERT INTO measurement_types (name, slug, unit, description)
VALUES
    ('Crops',            'crops_pct',           '%', 'Share of the classified parcel whose most common class was crops.'),
    ('Trees',            'trees_pct',           '%', 'Share of the classified parcel whose most common class was trees.'),
    ('Grass and shrub',  'grass_shrub_pct',     '%', 'Share of the classified parcel whose most common class was grass or shrub and scrub.'),
    ('Built up',         'built_pct',           '%', 'Share of the classified parcel whose most common class was buildings and paved ground.'),
    ('Water',            'water_pct',           '%', 'Share of the classified parcel whose most common class was water or flooded vegetation.'),
    ('Bare ground',      'bare_pct',            '%', 'Share of the classified parcel whose most common class was bare ground or snow.'),
    ('Classified',       'landcover_valid_pct', '%', 'Share of the parcel with at least one cloud-free view in the window, so it has a class.');

INSERT INTO methodology_measurements (methodology_id, measurement_type_id, position)
SELECT m.id, mt.id, x.position
FROM (VALUES
    ('crops_pct',           1),
    ('trees_pct',           2),
    ('grass_shrub_pct',     3),
    ('built_pct',           4),
    ('water_pct',           5),
    ('bare_pct',            6),
    ('landcover_valid_pct', 7)
) AS x(measurement, position)
JOIN methodologies m ON m.slug = 'landcover-dynamic-world' AND m.version = '1.0.0'
JOIN measurement_types mt ON mt.slug = x.measurement;

INSERT INTO methodology_parameters (methodology_id, name, label, type, min_value, max_value, default_value)
SELECT m.id, 'min_valid_pct', 'Minimum classified pixels (%)', 'number', 0, 100, 50
FROM methodologies m
WHERE m.slug = 'landcover-dynamic-world' AND m.version = '1.0.0';
