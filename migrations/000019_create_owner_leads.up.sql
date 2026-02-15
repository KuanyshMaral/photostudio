-- Owner Leads Table
CREATE TABLE owner_leads (
    id BIGSERIAL PRIMARY KEY,
    
    -- Contact person
    contact_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    contact_position VARCHAR(100),
    
    -- Company info
    company_name VARCHAR(255) NOT NULL,
    bin VARCHAR(12),
    legal_address TEXT,
    website VARCHAR(500),
    
    -- Application details
    use_case TEXT,
    how_found_us VARCHAR(255),
    
    -- Lead management
    status VARCHAR(20) NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'contacted', 'qualified', 'converted', 'rejected', 'lost')),
    priority INT DEFAULT 0,
    assigned_to BIGINT REFERENCES users(id),
    notes TEXT,
    
    -- Follow-up
    last_contacted_at TIMESTAMPTZ,
    next_follow_up_at TIMESTAMPTZ,
    follow_up_count INT DEFAULT 0,
    
    -- Conversion
    converted_at TIMESTAMPTZ,
    converted_user_id BIGINT REFERENCES users(id),
    rejection_reason TEXT,
    
    -- UTM tracking
    source VARCHAR(100),
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    referrer_url VARCHAR(500),
    
    -- Metadata
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_owner_leads_status ON owner_leads(status);
CREATE INDEX idx_owner_leads_email ON owner_leads(contact_email);
CREATE INDEX idx_owner_leads_assigned_to ON owner_leads(assigned_to);
CREATE INDEX idx_owner_leads_created_at ON owner_leads(created_at DESC);
