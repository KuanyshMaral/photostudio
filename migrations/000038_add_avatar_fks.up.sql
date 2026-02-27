-- 000038_add_avatar_fks.up.sql
-- Phase: Add 1:1 FK columns for single-file upload relationships.
-- Rule: 1:1 → FK column on the owning entity (no intermediate table needed).
-- Avatar URL is resolved dynamically by JOINing uploads at query time.

-- User avatar
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_upload_id VARCHAR(36)
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_avatar_upload_id
    ON users(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

COMMENT ON COLUMN users.avatar_upload_id IS
    '1:1 FK to uploads. NULL = no avatar set. ON DELETE SET NULL so deleting the file clears the avatar automatically.';
