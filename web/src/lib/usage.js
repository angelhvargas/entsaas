/**
 * Usage display and formatting utilities.
 */

export const USAGE_STATUS = Object.freeze({
  WITHIN_LIMIT: 'within_limit',
  NEAR_LIMIT: 'near_limit',
  AT_LIMIT: 'at_limit',
  OVER_LIMIT: 'over_limit',
  UNLIMITED: 'unlimited',
})

export const SEVERITY = Object.freeze({
  OK: 'ok',
  WARNING: 'warning',
  ERROR: 'error',
  NEUTRAL: 'neutral',
})

/**
 * Returns user-friendly status labels.
 */
export function usageStatusLabel(status) {
  switch (status) {
    case USAGE_STATUS.WITHIN_LIMIT: return 'Within limit'
    case USAGE_STATUS.NEAR_LIMIT: return 'Near limit'
    case USAGE_STATUS.AT_LIMIT: return 'At limit'
    case USAGE_STATUS.OVER_LIMIT: return 'Over limit'
    case USAGE_STATUS.UNLIMITED: return 'Unlimited'
    default: return status || 'Unknown'
  }
}

/**
 * Maps usage status to theme severity colors.
 */
export function usageStatusSeverity(status) {
  switch (status) {
    case USAGE_STATUS.WITHIN_LIMIT:
    case USAGE_STATUS.UNLIMITED:
      return SEVERITY.OK
    case USAGE_STATUS.NEAR_LIMIT:
      return SEVERITY.WARNING
    case USAGE_STATUS.AT_LIMIT:
    case USAGE_STATUS.OVER_LIMIT:
      return SEVERITY.ERROR
    default:
      return SEVERITY.NEUTRAL
  }
}

/**
 * Formats resource counters.
 */
export function formatUsage(current, limit) {
  if (limit === -1 || limit === undefined) return `${current} / Unlimited`
  return `${current} / ${limit}`
}

/**
 * Computes safe, clamped percentage.
 */
export function usagePercent(current, limit) {
  if (limit === -1 || limit === undefined || limit <= 0) return 0
  return Math.min(100, Math.round((current / limit) * 100))
}
