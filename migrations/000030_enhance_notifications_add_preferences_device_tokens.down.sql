-- Rollback Phase 3: Drop device_tokens table
DROP TABLE IF EXISTS device_tokens CASCADE;
DROP INDEX IF EXISTS idx_device_tokens_user_id;
DROP INDEX IF EXISTS idx_device_tokens_active;
DROP INDEX IF EXISTS idx_device_tokens_last_used;

-- Rollback Phase 2: Drop user_notification_preferences table
DROP TABLE IF EXISTS user_notification_preferences CASCADE;
DROP INDEX IF EXISTS idx_user_notification_preferences_user_id;

-- Rollback Phase 1: Revert notifications table changes
-- Safely rename body back to message if body exists and message doesn't
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'body'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'notifications' AND column_name = 'message'
    ) THEN
        ALTER TABLE notifications RENAME COLUMN body TO message;
    END IF;
END $$;

-- Remove read_at column if it exists
ALTER TABLE notifications DROP COLUMN IF EXISTS read_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_user_unread;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_type;
