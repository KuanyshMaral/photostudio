-- 000040_normalize_profiles_and_cleanup_deprecated.down.sql
-- Reverts: adds back profile fields to users, renames _deprecated columns back, removes avatar cols from profiles.

-- ─── Step 1: Restore _deprecated columns ────────────────────────────────────
ALTER TABLE studios
    ADD COLUMN IF NOT EXISTS photos_deprecated TEXT[];

ALTER TABLE rooms
    ADD COLUMN IF NOT EXISTS photos_deprecated TEXT[];

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS photos_deprecated TEXT[];

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS upload_id_deprecated VARCHAR(255);

-- ─── Step 2: Restore user profile columns ───────────────────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS name             VARCHAR(255),
    ADD COLUMN IF NOT EXISTS phone            VARCHAR(20),
    ADD COLUMN IF NOT EXISTS avatar_url       VARCHAR(500),
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_avatar_upload_id
    ON users(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

-- ─── Step 3: Restore backfilled data from client_profiles → users ────────────
UPDATE users u
SET    avatar_url       = cp.avatar_url,
       avatar_upload_id = cp.avatar_upload_id
FROM   client_profiles cp
WHERE  cp.user_id = u.id;

-- ─── Step 4: Remove avatar columns from owner_profiles and admin_profiles ────
DROP INDEX IF EXISTS idx_owner_profiles_avatar_upload_id;
ALTER TABLE owner_profiles
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS avatar_upload_id;

DROP INDEX IF EXISTS idx_admin_profiles_avatar_upload_id;
ALTER TABLE admin_profiles
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS avatar_upload_id;

DROP INDEX IF EXISTS idx_client_profiles_avatar_upload_id;
ALTER TABLE client_profiles
    DROP COLUMN IF EXISTS avatar_upload_id;
