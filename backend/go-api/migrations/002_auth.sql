-- ============================================================
-- Groundwork: Authentication
--
-- Users and the refresh tokens that keep their sessions alive.
-- Access tokens are signed JWTs and are never stored.
-- ============================================================


-- ============================================================
-- Table: users
-- ============================================================

CREATE TABLE users (

    -- Unique user identifier.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Login email, stored lowercased by the API.
    email TEXT NOT NULL UNIQUE,

    -- bcrypt hash of the password, never the password itself.
    password TEXT NOT NULL,

    first_name TEXT NOT NULL,

    last_name TEXT NOT NULL,

    -- Public handle.
    username TEXT NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Reuses set_updated_at() from 001_initial_schema.sql.
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- Table: refresh_tokens
--
-- One row per issued refresh token. Rows are revoked, not
-- deleted, when rotated so reuse of an old token is detectable.
-- ============================================================

CREATE TABLE refresh_tokens (

    -- Unique token identifier.
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Owner. Deleting a user removes their sessions.
    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    -- SHA-256 hex of the raw token. The raw token is only ever
    -- held by the client.
    token_hash TEXT NOT NULL UNIQUE,

    expires_at TIMESTAMPTZ NOT NULL,

    -- NULL while the token is still usable.
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX refresh_tokens_user_id_idx
    ON refresh_tokens (user_id);
