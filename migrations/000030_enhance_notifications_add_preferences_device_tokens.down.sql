-- Drop device_tokens table
DROP TABLE IF EXISTS device_tokens CASCADE;
DROP INDEX IF EXISTS idx_device_tokens_user_id;
DROP INDEX IF EXISTS idx_device_tokens_active;
DROP INDEX IF EXISTS idx_device_tokens_last_used;

-- Drop user_notification_preferences table
DROP TABLE IF EXISTS user_notification_preferences CASCADE;
DROP INDEX IF EXISTS idx_user_notification_preferences_user_id;

-- Remove read_at column from notifications
ALTER TABLE IF EXISTS notifications DROP COLUMN IF EXISTS read_at;

-- Safely rename body back to message if needed
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name='notifications' AND column_name='body'
  ) THEN
    ALTER TABLE notifications RENAME COLUMN body TO message;
  END IF;
END $$;
