CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    booking_id BIGINT NULL REFERENCES bookings(id) ON DELETE SET NULL,
    subscription_id UUID NULL,
    robokassa_invoice_id BIGINT NOT NULL UNIQUE,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    status VARCHAR(20) NOT NULL CHECK (status IN ('created','paid','failed')),
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    paid_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_booking_id ON payments(booking_id);
CREATE INDEX IF NOT EXISTS idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

CREATE TABLE IF NOT EXISTS recurring_subscriptions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending','active','canceled','expired')),
    first_invoice_id BIGINT UNIQUE,
    next_billing_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_subscriptions_first_invoice FOREIGN KEY (first_invoice_id) REFERENCES payments(robokassa_invoice_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_recurring_subscriptions_user_id ON recurring_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_subscriptions_status ON recurring_subscriptions(status);
