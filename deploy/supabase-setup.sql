-- One-time setup for a Supabase database, run in the Supabase SQL editor BEFORE the first
-- deploy (the deploy then runs the migrations). Safe to run again.

-- PostGIS must live in public: the queue functions are SECURITY DEFINER with
-- search_path = public, pg_temp, and call ST_AsGeoJSON. Supabase's default is the
-- extensions schema, which those functions can't see.
CREATE EXTENSION IF NOT EXISTS postgis SCHEMA public;

-- Supabase publishes the public schema through its Data API to the anon and
-- authenticated roles, which would expose users, snapshots and everything else to anyone
-- with the project's public key. Groundwork's only door is the Go API, so take those
-- grants away now and for every table the migrations create later. Also switch the
-- Data API off in Project Settings > Data API.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon, authenticated;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM anon, authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM anon, authenticated;
