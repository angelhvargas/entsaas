-- +goose Up
-- EntSaaS: Billing schema — plans, plan_versions, subscriptions.
--
-- Design notes:
--   - plans: immutable catalog rows identified by slug.
--   - plan_versions: versioned entitlements (JSONB). -1 = unlimited, 0 = unavailable.
--     provider_price_id is the external billing processor price identifier
--     (Stripe price_xxx or Paddle pri_xxx). NULL for free / enterprise sales-led.
--   - subscriptions: one active subscription per org at a time.
--     provider_subscription_id is kept for webhook reconciliation.
--
-- Capacity convention:
--   -1 = unlimited   0 = not available on this plan   N = hard limit
--
-- This migration is IDEMPOTENT:
--   plans use ON CONFLICT (slug) DO NOTHING
--   plan_versions use ON CONFLICT (plan_id, version) DO UPDATE SET

-- +goose StatementBegin

-- ── Plans ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS plans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug         TEXT NOT NULL UNIQUE,               -- immutable identifier
    display_name TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_plans_slug ON plans (slug);

-- ── Plan Versions (entitlements) ─────────────────────────────────────────────
-- Each version captures a point-in-time snapshot of what a plan includes.
-- Orgs on a subscription inherit the entitlements from the version they signed up on.
CREATE TABLE IF NOT EXISTS plan_versions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id           UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    version           INT  NOT NULL DEFAULT 1,
    notes             TEXT,
    entitlements      JSONB NOT NULL DEFAULT '{}',   -- see capacity convention above
    provider_price_id TEXT,                           -- Stripe price_xxx or Paddle pri_xxx
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, version)
);
CREATE INDEX IF NOT EXISTS idx_plan_versions_plan_id ON plan_versions (plan_id);

-- ── Subscriptions ────────────────────────────────────────────────────────────
-- One row per org (upserted on every lifecycle event from the billing webhook).
CREATE TABLE IF NOT EXISTS subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                  TEXT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id                 UUID NOT NULL REFERENCES plans(id),
    plan_version_id         UUID REFERENCES plan_versions(id),
    status                  TEXT NOT NULL DEFAULT 'active',  -- active | trialing | past_due | canceled | paused
    provider_subscription_id TEXT,                           -- Stripe sub_xxx or Paddle sub_xxx
    provider_customer_id    TEXT,                           -- Stripe cus_xxx or Paddle cus_xxx
    current_period_start    TIMESTAMPTZ,
    current_period_end      TIMESTAMPTZ,
    cancel_at_period_end    BOOLEAN NOT NULL DEFAULT false,
    trial_end_at            TIMESTAMPTZ,
    canceled_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_id        ON subscriptions (org_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_provider_sub  ON subscriptions (provider_subscription_id);

-- +goose StatementEnd

-- +goose StatementBegin

-- ── Seed canonical plan catalog ───────────────────────────────────────────────
-- Plans are idempotent on slug. Entitlements use DO UPDATE SET to stay in sync
-- with internal/billing/provider.go NoopProvider and any real catalog.

INSERT INTO plans (slug, display_name, description, is_active)
VALUES
  ('free',       'Free',       'Free forever — up to 3 projects, 3 team members. No credit card required.', true),
  ('pro',        'Pro',        'Pro — unlimited projects, AI assistant, priority support. $49/mo.',          true),
  ('enterprise', 'Enterprise', 'Enterprise — custom limits, SSO, dedicated SLA. Contact us.',               true)
ON CONFLICT (slug) DO NOTHING;

-- Free entitlements
INSERT INTO plan_versions (plan_id, version, notes, entitlements)
SELECT id, 1,
       'Seed v1 — Free tier (migration 003)',
       '{
         "max_projects": 3,
         "max_members": 3,
         "ai_assistant_enabled": false,
         "sso_enabled": false,
         "priority_support": false,
         "audit_log_retention_days": 30
       }'::jsonb
FROM plans WHERE slug = 'free'
ON CONFLICT (plan_id, version) DO UPDATE SET
  entitlements = EXCLUDED.entitlements,
  notes        = EXCLUDED.notes,
  updated_at   = NOW();

-- Pro entitlements
INSERT INTO plan_versions (plan_id, version, notes, entitlements)
SELECT id, 1,
       'Seed v1 — Pro tier $49/mo (migration 003)',
       '{
         "max_projects": -1,
         "max_members": -1,
         "ai_assistant_enabled": true,
         "sso_enabled": false,
         "priority_support": true,
         "audit_log_retention_days": 365
       }'::jsonb
FROM plans WHERE slug = 'pro'
ON CONFLICT (plan_id, version) DO UPDATE SET
  entitlements = EXCLUDED.entitlements,
  notes        = EXCLUDED.notes,
  updated_at   = NOW();

-- Enterprise entitlements
INSERT INTO plan_versions (plan_id, version, notes, entitlements)
SELECT id, 1,
       'Seed v1 — Enterprise tier, custom pricing (migration 003)',
       '{
         "max_projects": -1,
         "max_members": -1,
         "ai_assistant_enabled": true,
         "sso_enabled": true,
         "priority_support": true,
         "audit_log_retention_days": -1
       }'::jsonb
FROM plans WHERE slug = 'enterprise'
ON CONFLICT (plan_id, version) DO UPDATE SET
  entitlements = EXCLUDED.entitlements,
  notes        = EXCLUDED.notes,
  updated_at   = NOW();

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plan_versions;
DROP TABLE IF EXISTS plans;
