CREATE TABLE equipment (
                           id              BIGSERIAL PRIMARY KEY,
                           room_id         BIGINT NOT NULL
                               REFERENCES rooms(id) ON DELETE CASCADE,

                           name            VARCHAR(255) NOT NULL,
                           category        VARCHAR(100),  -- Camera, Lighting, Audio
                           brand           VARCHAR(100),
                           model           VARCHAR(100),
                           quantity        INTEGER DEFAULT 1 CHECK (quantity > 0),

                           rental_price    DECIMAL(10, 2) DEFAULT 0,  -- Если отдельно платно

                           created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_equipment_room
    ON equipment(room_id);
