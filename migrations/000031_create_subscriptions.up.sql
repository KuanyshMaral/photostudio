-- Subscription plans (tiers available on the platform)
-- Only Studio Owners subscribe. Clients are NOT affected by this system.
CREATE TABLE IF NOT EXISTS subscription_plans (
    id              VARCHAR(50) PRIMARY KEY,          -- 'free', 'starter', 'pro'
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    price_monthly   DECIMAL(10, 2) NOT NULL DEFAULT 0,
    price_yearly    DECIMAL(10, 2),

    -- Numeric limits for Studio Owners
    max_rooms           INT NOT NULL DEFAULT 1,       -- Max rooms a studio can create
    max_photos_per_room INT NOT NULL DEFAULT 5,       -- Max photos per room
    max_team_members    INT NOT NULL DEFAULT 0,       -- 0 = solo owner only

    -- Feature flags
    analytics_advanced  BOOLEAN NOT NULL DEFAULT FALSE, -- Access to advanced analytics
    priority_search     BOOLEAN NOT NULL DEFAULT FALSE, -- Boosted position in catalog
    priority_support    BOOLEAN NOT NULL DEFAULT FALSE, -- Priority email/phone support
    crm_access          BOOLEAN NOT NULL DEFAULT FALSE, -- Access to CRM/lead features

    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Seed default plans
INSERT INTO subscription_plans (id, name, description, price_monthly, price_yearly, max_rooms, max_photos_per_room, max_team_members, analytics_advanced, priority_search, priority_support, crm_access)
VALUES
    ('free',    'Бесплатный', 'Базовый план для старта',          0,      0,      1,  5,  0,  FALSE, FALSE, FALSE, FALSE),
    ('starter', 'Стартер',    'Для небольших студий',             9900,   99000,  3,  10, 2,  FALSE, FALSE, FALSE, TRUE),
    ('pro',     'Про',        'Для профессиональных студий',      19900,  199000, -1, 20, 10, TRUE,  TRUE,  TRUE,  TRUE)
ON CONFLICT (id) DO NOTHING;

-- Active subscriptions for Studio Owners
-- Clients (role='client') are NEVER referenced here.
CREATE TABLE IF NOT EXISTS subscriptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id             VARCHAR(50) NOT NULL REFERENCES subscription_plans(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'cancelled', 'expired', 'past_due', 'pending')),
    billing_period      VARCHAR(10) NOT NULL DEFAULT 'monthly'
                            CHECK (billing_period IN ('monthly', 'yearly')),
    started_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMP,
    auto_renew          BOOLEAN NOT NULL DEFAULT TRUE,
    cancel_reason       TEXT,
    cancelled_at        TIMESTAMP,
    payment_method_id   VARCHAR(255),                -- Stripe/Robokassa reference
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_owner_id ON subscriptions(owner_id);
CREATE INDEX idx_subscriptions_status   ON subscriptions(status);
