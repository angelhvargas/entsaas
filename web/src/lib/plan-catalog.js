/**
 * Shared Plan Catalog — Tier ordering and plan name helpers.
 *
 * Ported from assumetr EPIC-058. Provides a single source of truth for
 * tier ordering used by upgrade/downgrade button logic in PlanUsageView.
 *
 * NOTE: Full plan data (features, pricing, descriptions) is driven from the
 * API via the billing store's catalogPlans. This module only provides
 * structural constants that the API doesn't expose.
 *
 * @module lib/plan-catalog
 */

/**
 * Tier ordering for upgrade/downgrade detection.
 * Higher value = higher tier. Must match backend plan slugs.
 */
export const TIER_ORDER = { free: 0, pro: 1, enterprise: 2 }

/**
 * Get the human-readable name for a plan slug.
 * Falls back to the raw slug if not found.
 * @param {string} slug
 * @returns {string}
 */
export function getPlanName(slug) {
  const names = { free: 'Free', pro: 'Pro', enterprise: 'Enterprise' }
  return names[slug] ?? slug
}
