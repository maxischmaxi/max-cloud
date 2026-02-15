ALTER TABLE services ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id);

DROP INDEX IF EXISTS idx_services_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_org_name ON services (org_id, name);
