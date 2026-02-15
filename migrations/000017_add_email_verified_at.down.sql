-- Rollback email_verified_at addition
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
