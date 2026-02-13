ALTER TABLE users
    ADD COLUMN mwork_user_id UUID NULL;

ALTER TABLE users
    ADD COLUMN mwork_role TEXT NULL;

COMMENT ON COLUMN users.mwork_user_id IS 'Link to MWork user id';
COMMENT ON COLUMN users.mwork_role IS 'MWork user role';

CREATE INDEX idx_users_mwork_user_id ON users(mwork_user_id);
CREATE UNIQUE INDEX idx_users_mwork_user_id_unique
    ON users(mwork_user_id)
    WHERE mwork_user_id IS NOT NULL;
