CREATE TABLE IF NOT EXISTS pre_bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    studio_id BIGINT NOT NULL REFERENCES studios(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_pre_bookings_time CHECK (end_time > start_time),
    CONSTRAINT chk_pre_bookings_status CHECK (status IN ('pending', 'confirmed_unpaid', 'paid_confirmed', 'cancelled', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_pre_bookings_user_status ON pre_bookings (user_id, status);
CREATE INDEX IF NOT EXISTS idx_pre_bookings_studio_time ON pre_bookings (studio_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_pre_bookings_expires_at ON pre_bookings (expires_at);
