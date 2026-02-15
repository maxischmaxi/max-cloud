CREATE TABLE IF NOT EXISTS services (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    image      TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    url        TEXT NOT NULL DEFAULT '',
    env_vars   JSONB NOT NULL DEFAULT '{}',
    min_scale  INTEGER NOT NULL DEFAULT 0,
    max_scale  INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_services_name ON services (name);
