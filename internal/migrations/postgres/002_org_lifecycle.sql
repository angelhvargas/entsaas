-- +goose Up
-- EntSaaS: Organization Lifecycle — suspension and cooling-off deletion fields
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS suspended_reason TEXT;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS deletion_scheduled_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE organizations DROP COLUMN IF EXISTS suspended_at;
ALTER TABLE organizations DROP COLUMN IF EXISTS suspended_reason;
ALTER TABLE organizations DROP COLUMN IF EXISTS deletion_scheduled_at;
