BEGIN;

-- Drop device_tokens table
DROP TABLE IF EXISTS device_tokens CASCADE;

-- Drop user_notification_preferences table
DROP TABLE IF EXISTS user_notification_preferences CASCADE;

-- Remove read_at column from notifications
ALTER TABLE notifications DROP COLUMN IF EXISTS read_at;

-- Rename body back to message
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name = 'notifications' AND column_name = 'body'
  ) THEN
    ALTER TABLE notifications RENAME COLUMN body TO message;
  END IF;
END $$;

COMMIT;
