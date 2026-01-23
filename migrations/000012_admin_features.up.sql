-- VIP/Gold/Promo для студий
ALTER TABLE studios ADD COLUMN IF NOT EXISTS is_vip BOOLEAN DEFAULT FALSE;
ALTER TABLE studios ADD COLUMN IF NOT EXISTS is_gold BOOLEAN DEFAULT FALSE;
ALTER TABLE studios ADD COLUMN IF NOT EXISTS in_promo_slider BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_studios_promo ON studios(in_promo_slider);

-- Ads table
CREATE TABLE IF NOT EXISTS ads (
                                   id BIGSERIAL PRIMARY KEY,
                                   title VARCHAR(255) NOT NULL,
    image_url VARCHAR(500) NOT NULL,
    target_url VARCHAR(500),
    placement VARCHAR(50) NOT NULL DEFAULT 'home_banner',
    is_active BOOLEAN DEFAULT TRUE,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    impressions BIGINT DEFAULT 0,
    clicks BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX IF NOT EXISTS idx_ads_placement ON ads(placement);
CREATE INDEX IF NOT EXISTS idx_ads_active ON ads(is_active);

-- (важно для твоего кода booking_repo.go) deposit_amount и cancellation_reason
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS deposit_amount DECIMAL(10,2) DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;
