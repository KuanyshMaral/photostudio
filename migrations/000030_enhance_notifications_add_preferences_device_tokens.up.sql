-- Phase 1: Enhance notifications table
-- Add read_at column if it doesn't exist
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

-- Safely handle body column: rename if message exists, create if neither exists
DO $$
BEGIN
    -- Check if 'message' column exists and rename to 'body' if 'body' doesn't exist
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'message'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'body'
    ) THEN
        ALTER TABLE notifications RENAME COLUMN message TO body;
    END IF;
    
    -- If neither message nor body exist, add body column
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'body'
    ) THEN
        ALTER TABLE notifications ADD COLUMN body TEXT;
    END IF;
END $$;

-- Ensure data column exists and is JSONB type
DO $$
BEGIN
    -- If data column doesn't exist, create it as JSONB
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'data'
    ) THEN
        ALTER TABLE notifications ADD COLUMN data JSONB DEFAULT '{}'::jsonb;
    ELSE
        -- If it exists but is not JSONB, try to convert (wrapped in BEGIN/EXCEPTION to handle errors gracefully)
        BEGIN
            ALTER TABLE notifications ALTER COLUMN data TYPE jsonb USING data::jsonb;
        EXCEPTION WHEN OTHERS THEN
            -- If conversion fails, just leave it as is
            NULL;
        END;
    END IF;
END $$;

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);

-- Update comments
COMMENT ON TABLE notifications IS 'In-app уведомления пользователей с поддержкой read_at tracking';
COMMENT ON COLUMN notifications.body IS 'Текст уведомления';
COMMENT ON COLUMN notifications.read_at IS 'Время когда уведомление было прочитано';
COMMENT ON COLUMN notifications.data IS 'JSON структурированные данные связанные с уведомлением';

-- Phase 2: Create user_notification_preferences table
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    email_enabled BOOLEAN DEFAULT TRUE,
    push_enabled BOOLEAN DEFAULT TRUE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    digest_enabled BOOLEAN DEFAULT TRUE,
    digest_frequency VARCHAR(50) DEFAULT 'weekly',
    per_type_settings JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_user_id ON user_notification_preferences(user_id);

-- Phase 3: Create device_tokens table for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    platform VARCHAR(50) NOT NULL,
    device_name VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_active ON device_tokens(user_id, is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_device_tokens_last_used ON device_tokens(last_used_at);
