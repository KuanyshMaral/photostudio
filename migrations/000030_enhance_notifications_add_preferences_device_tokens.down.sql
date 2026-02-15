-- Rollback Phase 3: Drop device_tokens table
DROP TABLE IF EXISTS device_tokens CASCADE;
DROP INDEX IF EXISTS idx_device_tokens_user_id;
DROP INDEX IF EXISTS idx_device_tokens_active;
DROP INDEX IF EXISTS idx_device_tokens_last_used;

-- Rollback Phase 2: Drop user_notification_preferences table
DROP TABLE IF EXISTS user_notification_preferences CASCADE;
DROP INDEX IF EXISTS idx_user_notification_preferences_user_id;

-- Rollback Phase 1: Revert notifications table changes
-- Rename body back to message (if possible)
ALTER TABLE notifications RENAME COLUMN IF EXISTS body TO message;

-- Remove read_at column
ALTER TABLE notifications DROP COLUMN IF EXISTS read_at;

-- Drop new indexes
DROP INDEX IF EXISTS idx_notifications_user_unread;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_type;

-- Recreate old indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications(user_id, is_read)
    WHERE is_read = FALSE;

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications(user_id, created_at DESC);
