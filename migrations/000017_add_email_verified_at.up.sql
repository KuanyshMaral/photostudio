-- Add email_verified_at timestamp
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- Remove profile-related fields (moving to profile tables)
-- Note: We'll migrate the data first, then drop columns in a separate migration
-- This is a preparation migration to add the timestamp field
