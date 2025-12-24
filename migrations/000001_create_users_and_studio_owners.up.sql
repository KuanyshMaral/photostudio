CREATE TABLE users (
                       id BIGSERIAL PRIMARY KEY,

                       email VARCHAR(255) NOT NULL UNIQUE,
                       password_hash VARCHAR(255) NOT NULL,

                       role VARCHAR(20) NOT NULL
                           CHECK (role IN ('client', 'studio_owner', 'admin')),

                       name VARCHAR(255) NOT NULL,
                       phone VARCHAR(20),
                       avatar_url VARCHAR(500),

                       email_verified BOOLEAN DEFAULT false,

                       studio_status VARCHAR(20)
                           CHECK (studio_status IN ('pending', 'verified', 'rejected', 'blocked')),

                       created_at TIMESTAMPTZ DEFAULT NOW(),
                       updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email ON users(LOWER(email));
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_studio_status
    ON users(studio_status)
    WHERE role = 'studio_owner';

CREATE TABLE studio_owners (
                               id BIGSERIAL PRIMARY KEY,
                               user_id BIGINT NOT NULL UNIQUE
                                   REFERENCES users(id) ON DELETE CASCADE,

                               company_name VARCHAR(255) NOT NULL,
                               bin VARCHAR(12),
                               legal_address TEXT,
                               contact_person VARCHAR(255),
                               contact_position VARCHAR(100),

                               verification_docs TEXT[],
                               verified_at TIMESTAMPTZ,
                               verified_by BIGINT REFERENCES users(id),
                               rejected_reason TEXT,
                               admin_notes TEXT,

                               created_at TIMESTAMPTZ DEFAULT NOW()
);
