-- +goose Up
-- EntSaaS: Core schema — organizations, users, credentials, tokens, projects, audit

-- ── Organizations ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations (slug);

-- ── Users ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id                TEXT PRIMARY KEY,
    email             TEXT NOT NULL,
    role              TEXT NOT NULL DEFAULT 'member',
    org_id            TEXT NOT NULL REFERENCES organizations(id),
    is_active         BOOLEAN NOT NULL DEFAULT true,
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, email)
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users (org_id);

-- ── Credentials ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_credentials (
    user_id       TEXT PRIMARY KEY REFERENCES users(id),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Refresh Tokens ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token_hash  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    org_id      TEXT NOT NULL REFERENCES organizations(id),
    user_agent  TEXT,
    ip_address  TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- ── Reset Tokens ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reset_tokens (
    token_hash  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    used_at     TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 hour'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Verification Tokens ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS verification_tokens (
    token_hash  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    used_at     TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Invites ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS invites (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL REFERENCES organizations(id),
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_by  TEXT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invites_org_id ON invites (org_id);

-- ── Projects ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    org_id      TEXT NOT NULL REFERENCES organizations(id),
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_projects_org_id ON projects (org_id);

-- ── Audit Log ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    TEXT,
    org_id      TEXT NOT NULL,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_log_org_id ON audit_log (org_id, created_at DESC);

-- ── User Preferences ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id     TEXT PRIMARY KEY REFERENCES users(id),
    preferences JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS verification_tokens;
DROP TABLE IF EXISTS reset_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_credentials;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
