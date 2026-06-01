-- +goose Up
-- Add provider_customer_id to subscriptions.
--
-- Stripe's billing portal, payment method, and invoice APIs require a
-- customer ID (cus_xxx), not a subscription ID (sub_xxx). This column
-- stores the provider customer reference alongside the existing
-- provider_subscription_id.
--
-- For Paddle, the customer_id (ctm_xxx) is similarly needed for
-- transaction and subscription queries.
--
-- Populated during checkout linkage and webhook processing.

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS provider_customer_id TEXT;

COMMENT ON COLUMN subscriptions.provider_customer_id IS
    'Provider customer reference (e.g. Stripe cus_xxx, Paddle ctm_xxx). Set during checkout linkage.';

-- +goose Down
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS provider_customer_id;
