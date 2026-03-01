-- 000040_normalize_profiles_and_cleanup_deprecated.up.sql
-- Phase: Normalize DB - move avatar/name/phone from users -> per-role profile tables.
--        Drop all _deprecated columns. Table `users` becomes auth-only.

-- ─── Step 1: Add avatar columns to owner_profiles ────────────────────────────
ALTER TABLE owner_profiles
    ADD COLUMN IF NOT EXISTS avatar_url       VARCHAR(500),
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_owner_profiles_avatar_upload_id
    ON owner_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

-- ─── Step 1.5: Add missing avatar_upload_id to client_profiles ─────────────
ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_client_profiles_avatar_upload_id
    ON client_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

-- ─── Step 2: Add avatar columns to admin_profiles ────────────────────────────
ALTER TABLE admin_profiles
    ADD COLUMN IF NOT EXISTS avatar_url       VARCHAR(500),
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_admin_profiles_avatar_upload_id
    ON admin_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

-- ─── Step 3: Backfill avatars from users into profile tables ─────────────────
-- Copy avatar_url and avatar_upload_id from users into client_profiles
UPDATE client_profiles cp
SET    avatar_url       = u.avatar_url,
       avatar_upload_id = u.avatar_upload_id
FROM   users u
WHERE  cp.user_id = u.id
  AND  (u.avatar_url IS NOT NULL OR u.avatar_upload_id IS NOT NULL);

-- Copy avatar_url and avatar_upload_id from users into owner_profiles
UPDATE owner_profiles op
SET    avatar_url       = u.avatar_url,
       avatar_upload_id = u.avatar_upload_id
FROM   users u
WHERE  op.user_id = u.id
  AND  (u.avatar_url IS NOT NULL OR u.avatar_upload_id IS NOT NULL);

-- Copy avatar_url and avatar_upload_id from users into admin_profiles
-- admin_profiles.user_id is UUID, users.id is bigint - skip if using admin_users table
-- (admin_profiles is linked to admin_users, not regular users)

-- ─── Step 4: Drop user profile columns from users table ───────────────────────
-- These fields belong in the role-specific profile tables, not in `users`.
ALTER TABLE users
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS avatar_upload_id;

DROP INDEX IF EXISTS idx_users_avatar_upload_id;

-- ─── Step 5: Drop _deprecated columns (data already migrated to attachments) ──
ALTER TABLE studios
    DROP COLUMN IF EXISTS photos_deprecated;

ALTER TABLE rooms
    DROP COLUMN IF EXISTS photos_deprecated;

ALTER TABLE reviews
    DROP COLUMN IF EXISTS photos_deprecated;

ALTER TABLE messages
    DROP COLUMN IF EXISTS upload_id_deprecated;
