-- ============================================================
-- Groundwork: User roles
--
-- Auditors verify land claims; asset holders register the
-- parcels being verified.
-- ============================================================


CREATE TYPE user_role AS ENUM (
    'auditor',
    'asset_holder'
);


-- Existing accounts default to asset_holder. The default is then
-- dropped so the API must always choose a role explicitly.
ALTER TABLE users
    ADD COLUMN role user_role NOT NULL DEFAULT 'asset_holder';

ALTER TABLE users
    ALTER COLUMN role DROP DEFAULT;
