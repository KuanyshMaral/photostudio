CREATE TABLE rooms (
                       id                  BIGSERIAL PRIMARY KEY,
                       studio_id           BIGINT NOT NULL
                           REFERENCES studios(id) ON DELETE CASCADE,

                       name                VARCHAR(255) NOT NULL,
                       description         TEXT,

    -- Specs
                       area_sqm            INTEGER NOT NULL CHECK (area_sqm > 0),
                       capacity            INTEGER NOT NULL CHECK (capacity > 0),
                       room_type           VARCHAR(50) NOT NULL,  -- Fashion, Portrait, Creative

    -- Pricing
                       price_per_hour_min  DECIMAL(10, 2) NOT NULL CHECK (price_per_hour_min >= 0),
                       price_per_hour_max  DECIMAL(10, 2)
                           CHECK (price_per_hour_max >= price_per_hour_min),

    -- Amenities
                       amenities           TEXT[],  -- ['Wi-Fi', 'Паркинг', 'Кондиционер']
                       photos              TEXT[],  -- URLs

                       is_active           BOOLEAN DEFAULT true,
                       created_at          TIMESTAMPTZ DEFAULT NOW(),
                       updated_at          TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_rooms_studio ON rooms(studio_id) WHERE is_active = true;
CREATE INDEX idx_rooms_type ON rooms(room_type) WHERE is_active = true;
