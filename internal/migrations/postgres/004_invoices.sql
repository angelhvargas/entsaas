-- +goose Up
-- Invoice metadata table — persists provider invoice/transaction data locally
-- for audit, offline access, and provider-migration resilience.
--
-- The billing provider generates invoices; EntSaaS stores metadata locally
-- and proxies PDF access through signed URLs.
--
-- Design:
--   provider_id is the UNIQUE conflict key for idempotent upserts.
--   is_plan_change + plan_change_ctx track upgrade/downgrade invoices.
--   metadata JSONB stores provider-specific data without schema coupling.

CREATE TABLE IF NOT EXISTS invoices (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          TEXT        NOT NULL,
    provider_id     TEXT        NOT NULL,
    provider_name   TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'paid',
    amount_cents    INT         NOT NULL,
    currency        TEXT        NOT NULL DEFAULT 'usd',
    description     TEXT,
    invoice_number  TEXT,
    pdf_url         TEXT,
    hosted_url      TEXT,
    is_plan_change  BOOLEAN     NOT NULL DEFAULT false,
    plan_change_ctx TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata        JSONB
);

CREATE INDEX IF NOT EXISTS idx_invoices_org_id ON invoices(org_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_provider_id ON invoices(provider_id);

-- +goose Down
DROP TABLE IF EXISTS invoices;
