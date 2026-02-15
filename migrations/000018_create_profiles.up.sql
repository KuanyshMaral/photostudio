-- Client Profiles Table
CREATE TABLE client_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Basic info
    name VARCHAR(255),
    nickname VARCHAR(100),
    phone VARCHAR(20),
    avatar_url VARCHAR(500),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_client_profiles_user_id ON client_profiles(user_id);

-- Owner Profiles Table
CREATE TABLE owner_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Company info
    company_name VARCHAR(255) NOT NULL,
    bin VARCHAR(12),
    legal_address TEXT,
    contact_person VARCHAR(255),
    contact_position VARCHAR(100),
    
    -- Contact
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(500),
    
    -- Verification
    verification_status VARCHAR(20) DEFAULT 'pending'
        CHECK (verification_status IN ('pending', 'verified', 'rejected', 'blocked')),
    verification_docs TEXT[],
    verified_at TIMESTAMPTZ,
    verified_by BIGINT REFERENCES users(id),
    rejected_reason TEXT,
    admin_notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_owner_profiles_user_id ON owner_profiles(user_id);
CREATE INDEX idx_owner_profiles_verification ON owner_profiles(verification_status);

-- Admin Profiles Table
CREATE TABLE admin_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Admin info
    full_name VARCHAR(255) NOT NULL,
    position VARCHAR(100),
    phone VARCHAR(20),
    
    -- Access
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by BIGINT REFERENCES users(id),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_admin_profiles_user_id ON admin_profiles(user_id);
