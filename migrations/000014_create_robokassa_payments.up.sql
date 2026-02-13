CREATE TABLE IF NOT EXISTS robokassa_payments (
    id BIGSERIAL PRIMARY KEY,
    booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    out_sum VARCHAR(32) NOT NULL,
    inv_id BIGINT NOT NULL UNIQUE,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'created' CHECK (status IN ('created','pending','paid','failed')),
    signature VARCHAR(128),
    robokassa_url TEXT,
    shp_params TEXT,
    result_raw_body TEXT,
    success_raw_body TEXT,
    failure_reason TEXT,
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_robokassa_payments_booking_id ON robokassa_payments(booking_id);
CREATE INDEX IF NOT EXISTS idx_robokassa_payments_status ON robokassa_payments(status);
