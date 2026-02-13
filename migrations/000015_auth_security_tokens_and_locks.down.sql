DROP INDEX IF EXISTS idx_email_verif_user_used;
DROP INDEX IF EXISTS idx_email_verif_expires_at;
DROP TABLE IF EXISTS email_verification_codes;

DROP INDEX IF EXISTS idx_refresh_tokens_user_revoked;
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_family_id;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens;

ALTER TABLE users
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS failed_login_attempts,
    DROP COLUMN IF EXISTS ban_reason,
    DROP COLUMN IF EXISTS banned_at,
    DROP COLUMN IF EXISTS is_banned,
    DROP COLUMN IF EXISTS email_verified_at;
