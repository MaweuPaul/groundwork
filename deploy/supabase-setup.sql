-- One-time setup for a Supabase database, run in the Supabase SQL editor BEFORE the first
-- deploy (the deploy then runs the migrations). Safe to run again.

-- PostGIS must live in public: the queue functions are SECURITY DEFINER with
-- search_path = public, pg_temp, and call ST_AsGeoJSON. Supabase installs extensions into
-- the extensions schema, and PostGIS can't be moved with ALTER EXTENSION ... SET SCHEMA,
-- so an install elsewhere is dropped and recreated, but only while no column uses it.
DO $$
DECLARE
    current_schema_name TEXT;
BEGIN
    SELECT extnamespace::regnamespace::text INTO current_schema_name
    FROM pg_extension WHERE extname = 'postgis';

    IF current_schema_name IS NOT NULL AND current_schema_name <> 'public' THEN
        IF EXISTS (
            SELECT 1 FROM pg_attribute a
            JOIN pg_type t ON t.oid = a.atttypid
            JOIN pg_class c ON c.oid = a.attrelid
            WHERE t.typname IN ('geometry', 'geography')
              AND t.typnamespace::regnamespace::text = current_schema_name
              AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
              AND NOT a.attisdropped
              AND NOT EXISTS (
                  SELECT 1 FROM pg_depend d
                  WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
              )
        ) THEN
            RAISE EXCEPTION 'PostGIS is installed in schema % and columns already use it; move them before reinstalling it in public', current_schema_name;
        END IF;

        RAISE NOTICE 'Reinstalling PostGIS from schema % into public', current_schema_name;
        DROP EXTENSION postgis CASCADE;
    END IF;
END
$$;

CREATE EXTENSION IF NOT EXISTS postgis SCHEMA public;

-- Supabase's Data API serves the public schema to its anon and authenticated roles.
-- Groundwork's only door is the Go API, so the real protection is switching the Data API
-- off (Integrations > Data API, or untick it when creating the project).
--
-- These revokes are a second layer, but only partial: they remove grants made by the
-- postgres role, while Supabase's default privileges grant new tables to anon and
-- authenticated as supabase_admin, which postgres can't revoke. Those leftover grants are
-- unreachable while the Data API is off, because nothing else acts as those roles.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM anon, authenticated;
