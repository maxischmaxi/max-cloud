CREATE TABLE IF NOT EXISTS invitations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    role         TEXT NOT NULL CHECK (role IN ('admin', 'member')),
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),
    token_hash   TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    invited_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invitations_token_prefix ON invitations (token_prefix);
CREATE INDEX IF NOT EXISTS idx_invitations_org_id ON invitations (org_id);
