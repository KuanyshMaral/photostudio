-- Keep single source of truth for case-insensitive uniqueness:
-- UNIQUE(lower(email)). Drop plain UNIQUE(email) if present.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_email_key'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_key;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(LOWER(email));
