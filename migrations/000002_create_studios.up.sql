CREATE TABLE studios (
                         id              BIGSERIAL PRIMARY KEY,
                         owner_id        BIGINT NOT NULL
                             REFERENCES users(id) ON DELETE RESTRICT,

    -- Basic info
                         name            VARCHAR(255) NOT NULL,
                         description     TEXT,

    -- Location
                         address         VARCHAR(500) NOT NULL,
                         district        VARCHAR(100),
                         city            VARCHAR(100) DEFAULT 'Алматы',
                         latitude        DECIMAL(10, 8),
                         longitude       DECIMAL(11, 8),

    -- Rating (auto-calculated)
                         rating          DECIMAL(3, 2) DEFAULT 0.0 CHECK (rating BETWEEN 0 AND 5),
                         total_reviews   INTEGER DEFAULT 0,

    -- Contact
                         phone           VARCHAR(20),
                         email           VARCHAR(255),
                         website         VARCHAR(500),

    -- Working hours (flexible JSON)
                         working_hours   JSONB,

    -- Soft delete
                         deleted_at      TIMESTAMPTZ,

                         created_at      TIMESTAMPTZ DEFAULT NOW(),
                         updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_studios_owner ON studios(owner_id);
CREATE INDEX idx_studios_city ON studios(city) WHERE deleted_at IS NULL;
CREATE INDEX idx_studios_rating ON studios(rating DESC) WHERE deleted_at IS NULL;
