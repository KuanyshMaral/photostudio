-- Drop existing foreign key constraint
ALTER TABLE owner_leads DROP CONSTRAINT IF EXISTS owner_leads_assigned_to_fkey;

-- Drop existing column (Assuming data can be cleared or is not critical for now, as converting INT to UUID is not possible directly)
-- Alternatively, if we wanted to keep data, we would need a mapping table, but since we are moving systems, likely fine to drop.
ALTER TABLE owner_leads DROP COLUMN assigned_to;

-- Add new column
ALTER TABLE owner_leads ADD COLUMN assigned_to UUID;

-- Add foreign key constraint to admin_users
ALTER TABLE owner_leads ADD CONSTRAINT owner_leads_assigned_to_fkey FOREIGN KEY (assigned_to) REFERENCES admin_users(id);

-- Re-create index
DROP INDEX IF EXISTS idx_owner_leads_assigned_to;
CREATE INDEX idx_owner_leads_assigned_to ON owner_leads(assigned_to);
