-- Drop foreign key
ALTER TABLE owner_leads DROP CONSTRAINT IF EXISTS owner_leads_assigned_to_fkey;

-- Drop UUID column
ALTER TABLE owner_leads DROP COLUMN assigned_to;

-- Add back BIGINT column
ALTER TABLE owner_leads ADD COLUMN assigned_to BIGINT;

-- Add back foreign key to users (old admin user table)
ALTER TABLE owner_leads ADD CONSTRAINT owner_leads_assigned_to_fkey FOREIGN KEY (assigned_to) REFERENCES users(id);

-- Re-create index
DROP INDEX IF EXISTS idx_owner_leads_assigned_to;
CREATE INDEX idx_owner_leads_assigned_to ON owner_leads(assigned_to);
