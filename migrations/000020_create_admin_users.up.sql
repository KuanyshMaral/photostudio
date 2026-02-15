CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'support',
    name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed super admin
INSERT INTO admin_users (email, password_hash, role, name, is_active)
VALUES (
    'admin@photostudio.kz',
    '$2a$10$LQ4UCpPDWTyu00q/nOnAduAdJuF0ZF.D6a3.UgGmrDBkpLU9bjnbe', -- 'admin123'
    'super_admin',
    'Super Admin',
    true
);
