-- Phase 1: Enhance notifications table
-- Adding read_at to track when notification was read
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

-- Change message column to body for consistency
ALTER TABLE notifications RENAME COLUMN IF EXISTS message TO body;

-- Ensure data column is JSONB type
ALTER TABLE notifications ALTER COLUMN data TYPE jsonb USING data::jsonb;

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
    digest_frequency VARCHAR(50) DEFAULT 'weekly', -- daily, weekly, monthly
    per_type_settings JSONB DEFAULT '{}'::jsonb, -- Per-notification-type channel settings
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_user_id ON user_notification_preferences(user_id);

COMMENT ON TABLE user_notification_preferences IS 'Пользовательские предпочтения для уведомлений';
COMMENT ON COLUMN user_notification_preferences.per_type_settings IS 'Структура: {
  "notification_type": {
    "in_app": true,
    "email": false,
    "push": true
  }
}';

-- Phase 3: Create device_tokens table for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    platform VARCHAR(50) NOT NULL, -- web, ios, android
    device_name VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_active ON device_tokens(user_id, is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_device_tokens_last_used ON device_tokens(last_used_at);

COMMENT ON TABLE device_tokens IS 'Device tokens для отправки push-уведомлений';
COMMENT ON COLUMN device_tokens.platform IS 'Платформа: web, ios, android';
COMMENT ON COLUMN device_tokens.is_active IS 'Активен ли этот device token';

-- Drop old indexes if they exist from previous version and recreate them
DROP INDEX IF EXISTS idx_notifications_user_unread;
DROP INDEX IF EXISTS idx_notifications_user_created;

-- Recreate with new names
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
