-- ============================================================
-- Snapshot series
--
-- A series is one request split into windows (e.g. 1-month windows
-- every 3 months). Each window is an ordinary snapshot, computed and
-- sealed on its own. When every window has finished, the series is
-- sealed with a hash over the ordered list of window results, so the
-- set can't be cherry-picked: dropping or changing a window breaks it.
-- ============================================================
CREATE TABLE snapshot_series (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id TEXT UNIQUE,
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    methodology_id UUID NOT NULL REFERENCES methodologies(id),
    methodology_version TEXT NOT NULL,
    date_start DATE NOT NULL,
    date_end   DATE NOT NULL,
    window_months   INTEGER NOT NULL CHECK (window_months BETWEEN 1 AND 12),
    interval_months INTEGER NOT NULL CHECK (interval_months BETWEEN 1 AND 12),
    max_scene_cloud_pct   NUMERIC(5, 2) NOT NULL,
    cloud_score_threshold NUMERIC(4, 3) NOT NULL,
    -- Set once, when the last window finishes.
    record_hash TEXT UNIQUE,
    sealed_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (date_end >= date_start),
    CHECK ((record_hash IS NULL) = (sealed_at IS NULL))
);

CREATE TRIGGER snapshot_series_assign_short_id
BEFORE INSERT ON snapshot_series
FOR EACH ROW
EXECUTE FUNCTION assign_short_id();

ALTER TABLE snapshots
    ADD COLUMN series_id UUID REFERENCES snapshot_series(id);

CREATE INDEX snapshots_series_idx ON snapshots (series_id) WHERE series_id IS NOT NULL;


-- Series membership is an input too.
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
       OR NEW.max_scene_cloud_pct IS DISTINCT FROM OLD.max_scene_cloud_pct
       OR NEW.cloud_score_threshold IS DISTINCT FROM OLD.cloud_score_threshold
       OR NEW.record_format IS DISTINCT FROM OLD.record_format
       OR NEW.short_id IS DISTINCT FROM OLD.short_id
       OR NEW.series_id IS DISTINCT FROM OLD.series_id
    THEN
        RAISE EXCEPTION
            'Snapshot input and audit fields are immutable.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Series record text (format v1)
--
--   groundwork-series-v1
--   series_id, parcel_id, methodology_id, methodology_version,
--   date_start, date_end, window_months, interval_months,
--   max_scene_cloud_pct, cloud_score_threshold    (one per line, key=value)
--   window=<start>/<end>:<record_hash | failed>   (one per window, by start)
--   sealed_at=YYYY-MM-DDTHH:MI:SS.USZ
-- ============================================================
CREATE OR REPLACE FUNCTION series_record_text(p_series_id UUID, p_sealed_at TIMESTAMPTZ)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-series-v1',
        'series_id=' || r.id,
        'parcel_id=' || r.parcel_id,
        'methodology_id=' || r.methodology_id,
        'methodology_version=' || r.methodology_version,
        'date_start=' || to_char(r.date_start, 'YYYY-MM-DD'),
        'date_end=' || to_char(r.date_end, 'YYYY-MM-DD'),
        'window_months=' || r.window_months,
        'interval_months=' || r.interval_months,
        'max_scene_cloud_pct=' || r.max_scene_cloud_pct,
        'cloud_score_threshold=' || r.cloud_score_threshold,
        (
            SELECT string_agg(
                'window=' || to_char(s.date_start, 'YYYY-MM-DD') || '/' || to_char(s.date_end, 'YYYY-MM-DD')
                    || ':' || COALESCE(s.record_hash, 'failed'),
                E'\n' ORDER BY s.date_start
            )
            FROM snapshots s
            WHERE s.series_id = r.id
        ),
        'sealed_at=' || to_char(p_sealed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )
    FROM snapshot_series r
    WHERE r.id = p_series_id;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION series_record_hash(p_series_id UUID, p_sealed_at TIMESTAMPTZ)
RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(series_record_text(p_series_id, p_sealed_at), 'UTF8')), 'hex');
$$ LANGUAGE sql STABLE;


-- Seals the series when its last window completes or fails.
CREATE OR REPLACE FUNCTION seal_snapshot_series()
RETURNS TRIGGER AS $$
DECLARE
    now_ts TIMESTAMPTZ := now();
BEGIN
    -- Lock the series so two windows finishing together seal it once.
    PERFORM 1 FROM snapshot_series WHERE id = NEW.series_id AND record_hash IS NULL FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM snapshots
        WHERE series_id = NEW.series_id AND status NOT IN ('completed', 'failed')
    ) THEN
        RETURN NULL;
    END IF;

    UPDATE snapshot_series
    SET record_hash = series_record_hash(id, now_ts),
        sealed_at   = now_ts
    WHERE id = NEW.series_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;

CREATE TRIGGER snapshots_seal_series
AFTER UPDATE OF status ON snapshots
FOR EACH ROW
WHEN (NEW.series_id IS NOT NULL AND NEW.status IN ('completed', 'failed'))
EXECUTE FUNCTION seal_snapshot_series();


-- A sealed series can't change.
CREATE OR REPLACE FUNCTION freeze_sealed_series()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.record_hash IS NOT NULL THEN
        RAISE EXCEPTION 'Sealed series cannot be changed.';
    END IF;
    -- A BEFORE DELETE trigger that returns NULL silently cancels the delete.
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshot_series_freeze
BEFORE UPDATE OR DELETE ON snapshot_series
FOR EACH ROW
EXECUTE FUNCTION freeze_sealed_series();
