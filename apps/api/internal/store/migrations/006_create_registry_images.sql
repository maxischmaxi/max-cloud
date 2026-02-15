-- 006_create_registry_images.sql
-- Track registry images for cleanup and auditing purposes

CREATE TABLE IF NOT EXISTS registry_images (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    repository VARCHAR(511) NOT NULL,
    tags JSONB DEFAULT '[]',
    size_bytes BIGINT DEFAULT 0,
    digest VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_registry_images_org ON registry_images(org_id);
CREATE INDEX idx_registry_images_repo ON registry_images(repository);