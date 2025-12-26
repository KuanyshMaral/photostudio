CREATE TABLE IF NOT EXISTS bookings (
                                        id              BIGSERIAL PRIMARY KEY,

                                        room_id         BIGINT NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT,
                                        studio_id       BIGINT NOT NULL REFERENCES studios(id) ON DELETE RESTRICT,
                                        user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

                                        start_time      TIMESTAMPTZ NOT NULL,
                                        end_time        TIMESTAMPTZ NOT NULL CHECK (end_time > start_time),

                                        total_price     DECIMAL(10, 2) NOT NULL CHECK (total_price >= 0),

                                        status          VARCHAR(20) DEFAULT 'pending'
                                            CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed')),

                                        payment_status  VARCHAR(20) DEFAULT 'unpaid'
                                            CHECK (payment_status IN ('unpaid', 'paid', 'refunded')),

                                        notes           TEXT,

                                        created_at      TIMESTAMPTZ DEFAULT NOW(),
                                        updated_at      TIMESTAMPTZ DEFAULT NOW(),
                                        cancelled_at    TIMESTAMPTZ
);

-- CRITICAL: no overbooking
CREATE UNIQUE INDEX IF NOT EXISTS idx_no_overbooking ON bookings (
                                                                  room_id,
                                                                  tstzrange(start_time, end_time, '[)')
    ) WHERE status NOT IN ('cancelled');

-- Other indexes
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_room ON bookings(room_id);
CREATE INDEX IF NOT EXISTS idx_bookings_studio ON bookings(studio_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
