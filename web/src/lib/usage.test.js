import { describe, it, expect } from 'vitest'
import {
  USAGE_STATUS,
  SEVERITY,
  usageStatusLabel,
  usageStatusSeverity,
  formatUsage,
  usagePercent,
} from './usage'

describe('usage.js utilities', () => {
  describe('usageStatusLabel', () => {
    it('returns friendly labels for standard statuses', () => {
      expect(usageStatusLabel(USAGE_STATUS.WITHIN_LIMIT)).toBe('Within limit')
      expect(usageStatusLabel(USAGE_STATUS.NEAR_LIMIT)).toBe('Near limit')
      expect(usageStatusLabel(USAGE_STATUS.AT_LIMIT)).toBe('At limit')
      expect(usageStatusLabel(USAGE_STATUS.OVER_LIMIT)).toBe('Over limit')
      expect(usageStatusLabel(USAGE_STATUS.UNLIMITED)).toBe('Unlimited')
    })

    it('falls back to input string or Unknown', () => {
      expect(usageStatusLabel('custom-status')).toBe('custom-status')
      expect(usageStatusLabel(null)).toBe('Unknown')
    })
  })

  describe('usageStatusSeverity', () => {
    it('maps statuses to the correct theme severity levels', () => {
      expect(usageStatusSeverity(USAGE_STATUS.WITHIN_LIMIT)).toBe(SEVERITY.OK)
      expect(usageStatusSeverity(USAGE_STATUS.UNLIMITED)).toBe(SEVERITY.OK)
      expect(usageStatusSeverity(USAGE_STATUS.NEAR_LIMIT)).toBe(SEVERITY.WARNING)
      expect(usageStatusSeverity(USAGE_STATUS.AT_LIMIT)).toBe(SEVERITY.ERROR)
      expect(usageStatusSeverity(USAGE_STATUS.OVER_LIMIT)).toBe(SEVERITY.ERROR)
    })

    it('falls back to NEUTRAL', () => {
      expect(usageStatusSeverity('non-existent')).toBe(SEVERITY.NEUTRAL)
    })
  })

  describe('formatUsage', () => {
    it('formats normal resource limits', () => {
      expect(formatUsage(3, 5)).toBe('3 / 5')
      expect(formatUsage(0, 10)).toBe('0 / 10')
    })

    it('formats unlimited resources', () => {
      expect(formatUsage(12, -1)).toBe('12 / Unlimited')
      expect(formatUsage(5, undefined)).toBe('5 / Unlimited')
    })
  })

  describe('usagePercent', () => {
    it('computes correct percentages', () => {
      expect(usagePercent(2, 10)).toBe(20)
      expect(usagePercent(5, 5)).toBe(100)
    })

    it('clips percentages at 100%', () => {
      expect(usagePercent(15, 10)).toBe(100)
    })

    it('handles unlimited and division by zero/negative limits gracefully', () => {
      expect(usagePercent(5, -1)).toBe(0)
      expect(usagePercent(5, undefined)).toBe(0)
      expect(usagePercent(5, 0)).toBe(0)
      expect(usagePercent(5, -5)).toBe(0)
    })
  })
})
