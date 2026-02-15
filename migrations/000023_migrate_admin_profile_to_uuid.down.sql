-- Revert changes (Note: this is destructive to data because we dropped columns)
-- In a real scenario, we would need to map back to BIGINT IDs if they exist.

ALTER TABLE admin_profiles DROP CONSTRAINT IF EXISTS admin_profiles_user_id_fkey;
ALTER TABLE admin_profiles DROP CONSTRAINT IF EXISTS admin_profiles_created_by_fkey;

ALTER TABLE admin_profiles DROP COLUMN user_id;
ALTER TABLE admin_profiles DROP COLUMN created_by;

ALTER TABLE admin_profiles ADD COLUMN user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE admin_profiles ADD COLUMN created_by BIGINT REFERENCES users(id);

-- Re-create index
DROP INDEX IF EXISTS idx_admin_profiles_user_id;
CREATE INDEX idx_admin_profiles_user_id ON admin_profiles(user_id);
