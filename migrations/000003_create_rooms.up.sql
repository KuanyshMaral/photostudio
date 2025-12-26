CREATE TABLE rooms (
                       id BIGSERIAL PRIMARY KEY,
                       studio_id BIGINT NOT NULL
                           REFERENCES studios(id) ON DELETE CASCADE,

                       name VARCHAR(255) NOT NULL,
                       room_type VARCHAR(50) NOT NULL,

                       price_per_hour_min DECIMAL(10,2) NOT NULL CHECK (price_per_hour_min >= 0),
                       price_per_hour_max DECIMAL(10,2)
                           CHECK (price_per_hour_max >= price_per_hour_min),

                       is_active BOOLEAN DEFAULT true,

                       created_at TIMESTAMPTZ DEFAULT NOW(),
                       updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rooms_studio
    ON rooms(studio_id)
    WHERE is_active = true;
