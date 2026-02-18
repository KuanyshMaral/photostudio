-- Simple uploads table: tracks physical files stored on local disk.
-- No staging, no cloud, no 2-phase commit. Just: upload -> get ID -> use it.
CREATE TABLE IF NOT EXISTS uploads (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_name VARCHAR(255) NOT NULL,
    file_path     VARCHAR(500) NOT NULL,  -- relative: "2024/05/20/uuid_avatar.jpg"
    file_url      VARCHAR(500) NOT NULL,  -- public:   "/static/uploads/2024/05/20/uuid_avatar.jpg"
    mime_type     VARCHAR(100) NOT NULL,
    size          BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_uploads_user_id ON uploads(user_id);
