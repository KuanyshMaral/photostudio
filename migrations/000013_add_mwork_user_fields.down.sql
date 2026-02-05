DROP INDEX IF EXISTS idx_users_mwork_user_id_unique;
DROP INDEX IF EXISTS idx_users_mwork_user_id;

ALTER TABLE users DROP COLUMN IF EXISTS mwork_role;
ALTER TABLE users DROP COLUMN IF EXISTS mwork_user_id;
