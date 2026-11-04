-- ============================================================
-- Why a refresh token was revoked
--
-- 'rotated' tokens get a short reuse window, so parallel requests
-- don't sign the user out. A 'logout' token is dead at once.
-- ============================================================
ALTER TABLE refresh_tokens
    ADD COLUMN revoke_reason TEXT;

UPDATE refresh_tokens
SET revoke_reason = 'rotated'
WHERE revoked_at IS NOT NULL;

ALTER TABLE refresh_tokens
    ADD CONSTRAINT refresh_tokens_revoke_reason_check
        CHECK (revoke_reason IN ('rotated', 'logout')),
    ADD CONSTRAINT refresh_tokens_revoked_reason_pair_check
        CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL));
