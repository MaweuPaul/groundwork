-- ============================================================
-- Short public IDs for parcels and snapshots
--
-- URLs use them instead of UUIDs, e.g. /parcels/nyeri-block-a-k7x2p9qd.
-- The name part of that URL is cosmetic; the short ID is the key, so a
-- renamed parcel's old links keep working.
--
-- 8 characters of lowercase Crockford base32 (no i, l, o, u, so it reads
-- aloud and copies cleanly): 40 random bits. The database assigns them and
-- overwrites anything a client sends.
-- ============================================================

CREATE OR REPLACE FUNCTION gen_short_id()
RETURNS TEXT AS $$
DECLARE
    alphabet CONSTANT TEXT := '0123456789abcdefghjkmnpqrstvwxyz';
    -- Bytes 6 and 8 of a v4 UUID carry the version and variant bits, so they're skipped.
    positions CONSTANT INT[] := ARRAY[0, 1, 2, 3, 4, 5, 10, 11];
    bytes BYTEA := uuid_send(gen_random_uuid());
    result TEXT := '';
    pos INT;
BEGIN
    FOREACH pos IN ARRAY positions LOOP
        result := result || substr(alphabet, get_byte(bytes, pos) % 32 + 1, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Retries on the (very rare) clash; the unique constraint backs it up under concurrency.
CREATE OR REPLACE FUNCTION assign_short_id()
RETURNS TRIGGER AS $$
DECLARE
    taken BOOLEAN;
BEGIN
    LOOP
        NEW.short_id := gen_short_id();
        EXECUTE format('SELECT EXISTS (SELECT 1 FROM %I.%I WHERE short_id = $1)',
                       TG_TABLE_SCHEMA, TG_TABLE_NAME)
            INTO taken USING NEW.short_id;
        EXIT WHEN NOT taken;
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- A volatile default fills existing rows by rewriting the table, without firing
-- UPDATE triggers, so completed (frozen) snapshots get an ID too.
ALTER TABLE parcels ADD COLUMN short_id TEXT NOT NULL DEFAULT gen_short_id();
ALTER TABLE parcels ALTER COLUMN short_id DROP DEFAULT;
ALTER TABLE parcels
    ADD CONSTRAINT parcels_short_id_key UNIQUE (short_id),
    ADD CONSTRAINT parcels_short_id_format_check CHECK (short_id ~ '^[0-9a-hjkmnp-tv-z]{8}$');

CREATE TRIGGER parcels_assign_short_id
BEFORE INSERT ON parcels
FOR EACH ROW
EXECUTE FUNCTION assign_short_id();

ALTER TABLE snapshots ADD COLUMN short_id TEXT NOT NULL DEFAULT gen_short_id();
ALTER TABLE snapshots ALTER COLUMN short_id DROP DEFAULT;
ALTER TABLE snapshots
    ADD CONSTRAINT snapshots_short_id_key UNIQUE (short_id),
    ADD CONSTRAINT snapshots_short_id_format_check CHECK (short_id ~ '^[0-9a-hjkmnp-tv-z]{8}$');

-- 05 so it runs before the freeze and hash-chain triggers (they fire in name order).
CREATE TRIGGER snapshots_05_assign_short_id
BEFORE INSERT ON snapshots
FOR EACH ROW
EXECUTE FUNCTION assign_short_id();
