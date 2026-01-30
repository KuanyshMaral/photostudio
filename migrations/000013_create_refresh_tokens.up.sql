CREATE TABLE refresh_tokens (
                                id BIGSERIAL PRIMARY KEY,
                                user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token_hash CHAR(64) NOT NULL UNIQUE,
                                created_at TIMESTAMPTZ DEFAULT NOW(),
                                expires_at TIMESTAMPTZ NOT NULL,
                                revoked_at TIMESTAMPTZ,
                                replaced_by_id BIGINT
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at);
