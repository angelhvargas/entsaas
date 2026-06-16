-- 006_backfill_free_plan_subscriptions.sql
--
-- Backfill: assign free-plan subscription to all orgs without one.
-- Ported from assumetr EPIC-058 (101_backfill_free_plan_subscriptions.sql).
--
-- This is idempotent — INSERT ... WHERE NOT EXISTS.
-- Rollback marker: delete subscriptions created by this migration
-- by checking for plan_slug='free' and provider_subscription_id IS NULL.

-- +goose Up
INSERT INTO subscriptions (org_id, plan_id, plan_version_id, status, created_at, updated_at)
SELECT o.id, p.id, latest_pv.id, 'active', NOW(), NOW()
FROM organizations o
CROSS JOIN plans p
JOIN LATERAL (
    SELECT pv.id
    FROM plan_versions pv
    WHERE pv.plan_id = p.id
    ORDER BY pv.version DESC
    LIMIT 1
) latest_pv ON true
WHERE p.slug = 'free'
  AND NOT EXISTS (
    SELECT 1 FROM subscriptions s WHERE s.org_id = o.id
  );

-- +goose Down
-- Remove only backfilled free subs (no provider backing = synthetic).
DELETE FROM subscriptions
WHERE plan_id = (SELECT id FROM plans WHERE slug = 'free' LIMIT 1)
  AND provider_subscription_id IS NULL;
