-- Drop existing constraints
ALTER TABLE admin_profiles DROP CONSTRAINT IF EXISTS admin_profiles_user_id_fkey;
ALTER TABLE admin_profiles DROP CONSTRAINT IF EXISTS admin_profiles_created_by_fkey;

-- Drop existing columns
ALTER TABLE admin_profiles DROP COLUMN user_id;
ALTER TABLE admin_profiles DROP COLUMN created_by;

-- Add new UUID columns
ALTER TABLE admin_profiles ADD COLUMN user_id UUID NOT NULL UNIQUE;
ALTER TABLE admin_profiles ADD COLUMN created_by UUID;

-- Add foreign key constraints to admin_users
ALTER TABLE admin_profiles ADD CONSTRAINT admin_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES admin_users(id) ON DELETE CASCADE;
ALTER TABLE admin_profiles ADD CONSTRAINT admin_profiles_created_by_fkey FOREIGN KEY (created_by) REFERENCES admin_users(id);

-- Re-create index
DROP INDEX IF EXISTS idx_admin_profiles_user_id;
CREATE INDEX idx_admin_profiles_user_id ON admin_profiles(user_id);
