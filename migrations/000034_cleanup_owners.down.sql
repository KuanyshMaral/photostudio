CREATE TABLE studio_owners (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
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
    created_at TIMESTAMPTZ DEFAULT NOW(),
    org_type VARCHAR(20),
    legal_name VARCHAR(255)
);
