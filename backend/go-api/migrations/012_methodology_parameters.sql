-- ============================================================
-- Methodology parameters
--
-- The settings each methodology accepts, so the API and frontend
-- can read them. They mirror the plugin's manifest, and the worker
-- refuses to load a plugin whose manifest differs. Insert-only,
-- like methodologies: a new spec means a new methodology version.
-- ============================================================
CREATE TABLE methodology_parameters (
    methodology_id UUID NOT NULL REFERENCES methodologies(id),
    name           TEXT NOT NULL,
    label          TEXT NOT NULL,
    type           TEXT NOT NULL,
    min_value      NUMERIC NOT NULL,
    max_value      NUMERIC NOT NULL,
    default_value  NUMERIC NOT NULL,
    PRIMARY KEY (methodology_id, name),
    CHECK (min_value <= default_value AND default_value <= max_value)
);

CREATE TRIGGER methodology_parameters_prevent_update
BEFORE UPDATE ON methodology_parameters
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();

CREATE TRIGGER methodology_parameters_prevent_delete
BEFORE DELETE ON methodology_parameters
FOR EACH ROW
EXECUTE FUNCTION prevent_methodology_changes();

INSERT INTO methodology_parameters (methodology_id, name, label, type, min_value, max_value, default_value)
SELECT m.id, p.name, p.label, 'number', p.min_value, p.max_value, p.default_value
FROM methodologies m
CROSS JOIN (VALUES
    ('max_scene_cloud_pct',   'Max scene cloud cover (%)', 0.01, 100,  20),
    ('cloud_score_threshold', 'Clear pixel threshold',     0.40, 0.80, 0.60)
) AS p(name, label, min_value, max_value, default_value)
WHERE m.slug = 'ndvi-s2-median' AND m.version = '1.0.0';

GRANT SELECT ON methodology_parameters TO gee_worker;
