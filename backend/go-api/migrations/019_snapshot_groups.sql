-- ============================================================
-- Snapshot groups
--
-- A group is several methodologies run on one parcel for the same
-- window (e.g. NDVI and rainfall for one season), requested
-- together. Each member is an ordinary snapshot, computed and
-- sealed in the parcel's chain on its own. When every member has
-- finished, the group is sealed with a hash over its window and each
-- member's result, so the set can't be cherry-picked: dropping or
-- changing a member breaks it. Later analyses read from groups.
-- ============================================================
CREATE TABLE snapshot_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id   TEXT UNIQUE,
    parcel_id  UUID NOT NULL REFERENCES parcels(id),
    date_start DATE NOT NULL,
    date_end   DATE NOT NULL,
    record_format INTEGER NOT NULL DEFAULT 1 CHECK (record_format IN (1)),
    -- Set once, when the last member finishes.
    record_hash TEXT UNIQUE,
    sealed_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (date_end >= date_start),
    CHECK ((record_hash IS NULL) = (sealed_at IS NULL))
);

CREATE TRIGGER snapshot_groups_assign_short_id
BEFORE INSERT ON snapshot_groups
FOR EACH ROW
EXECUTE FUNCTION assign_short_id();

ALTER TABLE snapshots
    ADD COLUMN group_id UUID REFERENCES snapshot_groups(id),
    ADD CONSTRAINT snapshots_series_or_group CHECK (series_id IS NULL OR group_id IS NULL);

-- One snapshot per methodology in a group.
CREATE UNIQUE INDEX snapshots_group_methodology_idx ON snapshots (group_id, methodology_id) WHERE group_id IS NOT NULL;


-- A member has its group's parcel and window, and joins before the group is sealed.
CREATE OR REPLACE FUNCTION check_snapshot_group_member()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM snapshot_groups g
        WHERE g.id = NEW.group_id
          AND g.parcel_id = NEW.parcel_id
          AND g.date_start = NEW.date_start
          AND g.date_end = NEW.date_end
          AND g.record_hash IS NULL
    ) THEN
        RAISE EXCEPTION 'A group member must have its group''s parcel and dates, and join before the group is sealed.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshots_check_group_member
BEFORE INSERT ON snapshots
FOR EACH ROW
WHEN (NEW.group_id IS NOT NULL)
EXECUTE FUNCTION check_snapshot_group_member();


-- Group membership is an input too.
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
       OR NEW.parameters IS DISTINCT FROM OLD.parameters
       OR NEW.record_format IS DISTINCT FROM OLD.record_format
       OR NEW.short_id IS DISTINCT FROM OLD.short_id
       OR NEW.series_id IS DISTINCT FROM OLD.series_id
       OR NEW.group_id IS DISTINCT FROM OLD.group_id
    THEN
        RAISE EXCEPTION
            'Snapshot input and audit fields are immutable.';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ============================================================
-- Group record text (format v1)
--
--   groundwork-group-v1
--   group_id, parcel_id, date_start, date_end    (one per line, key=value)
--   member=<slug>@<version>:<record_hash | failed>  (one per member, by slug)
--   sealed_at=YYYY-MM-DDTHH:MI:SS.USZ
-- ============================================================
CREATE FUNCTION group_record_text(p_group_id UUID, p_sealed_at TIMESTAMPTZ)
RETURNS TEXT AS $$
    SELECT concat_ws(E'\n',
        'groundwork-group-v' || g.record_format,
        'group_id=' || g.id,
        'parcel_id=' || g.parcel_id,
        'date_start=' || to_char(g.date_start, 'YYYY-MM-DD'),
        'date_end=' || to_char(g.date_end, 'YYYY-MM-DD'),
        (
            SELECT string_agg(
                'member=' || m.slug || '@' || s.methodology_version || ':' || COALESCE(s.record_hash, 'failed'),
                E'\n' ORDER BY m.slug COLLATE "C", s.methodology_version COLLATE "C"
            )
            FROM snapshots s
            JOIN methodologies m ON m.id = s.methodology_id
            WHERE s.group_id = g.id
        ),
        'sealed_at=' || to_char(p_sealed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )
    FROM snapshot_groups g
    WHERE g.id = p_group_id;
$$ LANGUAGE sql STABLE;

CREATE FUNCTION group_record_hash(p_group_id UUID, p_sealed_at TIMESTAMPTZ)
RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(group_record_text(p_group_id, p_sealed_at), 'UTF8')), 'hex');
$$ LANGUAGE sql STABLE;


-- Seals the group when its last member completes or fails.
CREATE FUNCTION seal_snapshot_group()
RETURNS TRIGGER AS $$
DECLARE
    now_ts TIMESTAMPTZ := now();
BEGIN
    -- Lock the group so two members finishing together seal it once.
    PERFORM 1 FROM snapshot_groups WHERE id = NEW.group_id AND record_hash IS NULL FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM snapshots
        WHERE group_id = NEW.group_id AND status NOT IN ('completed', 'failed')
    ) THEN
        RETURN NULL;
    END IF;

    UPDATE snapshot_groups
    SET record_hash = group_record_hash(id, now_ts),
        sealed_at   = now_ts
    WHERE id = NEW.group_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;

CREATE TRIGGER snapshots_seal_group
AFTER UPDATE OF status ON snapshots
FOR EACH ROW
WHEN (NEW.group_id IS NOT NULL AND NEW.status IN ('completed', 'failed'))
EXECUTE FUNCTION seal_snapshot_group();


-- A sealed group can't change.
CREATE FUNCTION freeze_sealed_group()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.record_hash IS NOT NULL THEN
        RAISE EXCEPTION 'Sealed groups cannot be changed.';
    END IF;
    -- A BEFORE DELETE trigger that returns NULL silently cancels the delete.
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshot_groups_freeze
BEFORE UPDATE OR DELETE ON snapshot_groups
FOR EACH ROW
EXECUTE FUNCTION freeze_sealed_group();


-- ============================================================
-- verify_snapshot_group: recomputes a sealed group's hash. No rows
-- means it is intact, or not sealed yet.
-- ============================================================
CREATE FUNCTION verify_snapshot_group(p_group_id UUID)
RETURNS TABLE (problem TEXT) AS $$
    SELECT 'group record_hash does not match its members'
    FROM snapshot_groups g
    WHERE g.id = p_group_id
      AND g.record_hash IS NOT NULL
      AND g.record_hash IS DISTINCT FROM group_record_hash(g.id, g.sealed_at);
$$ LANGUAGE sql STABLE;
