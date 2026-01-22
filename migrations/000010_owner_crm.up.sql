-- Block 9: Owner CRM Tables

-- PIN коды владельцев
CREATE TABLE IF NOT EXISTS owner_pins (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pin_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Закупки
CREATE TABLE IF NOT EXISTS procurement_items (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INTEGER DEFAULT 1,
    est_cost DECIMAL(10,2),
    priority VARCHAR(20) DEFAULT 'medium',
    is_completed BOOLEAN DEFAULT FALSE,
    due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_procurement_owner ON procurement_items(owner_id);
CREATE INDEX idx_procurement_completed ON procurement_items(is_completed);

-- Обслуживание
CREATE TABLE IF NOT EXISTS maintenance_items (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to VARCHAR(255),
    due_date TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_maintenance_owner ON maintenance_items(owner_id);
CREATE INDEX idx_maintenance_status ON maintenance_items(status);

-- Block 12: Company Profile

CREATE TABLE IF NOT EXISTS company_profiles (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    logo VARCHAR(500),
    company_name VARCHAR(255),
    contact_person VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    website VARCHAR(255),
    city VARCHAR(100),
    company_type VARCHAR(50),
    description TEXT,
    specialization VARCHAR(255),
    years_experience INTEGER,
    team_size INTEGER,
    work_hours VARCHAR(100),
    services JSONB DEFAULT '[]',
    socials JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_company_owner ON company_profiles(owner_id);

-- Портфолио
CREATE TABLE IF NOT EXISTS portfolio_projects (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,
    title VARCHAR(255),
    category VARCHAR(100),
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_portfolio_owner ON portfolio_projects(owner_id);
