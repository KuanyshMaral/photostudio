CREATE TABLE studios (
                         id BIGSERIAL PRIMARY KEY,
                         owner_id BIGINT NOT NULL,

                         name VARCHAR(255) NOT NULL,
                         description TEXT,
                         address VARCHAR(500) NOT NULL,
                         city VARCHAR(100) DEFAULT 'Алматы',

                         rating DECIMAL(3,2) DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
                         total_reviews INT DEFAULT 0,

                         deleted_at TIMESTAMPTZ,
                         created_at TIMESTAMPTZ DEFAULT NOW(),
                         updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_studios_city
    ON studios(city)
    WHERE deleted_at IS NULL;
