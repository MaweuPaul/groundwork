-- The latest day each methodology's source has data for, so the API can refuse a window
-- the data doesn't reach yet (CHIRPS is published a few weeks after each month ends).
-- The worker refreshes it from Earth Engine; a methodology without a row has no known end.
-- Unlike the methodology tables this changes over time, so it isn't insert-only.
CREATE TABLE methodology_coverage (
    methodology_id UUID PRIMARY KEY REFERENCES methodologies (id),
    data_end DATE NOT NULL,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

GRANT SELECT, INSERT, UPDATE ON methodology_coverage TO gee_worker;
