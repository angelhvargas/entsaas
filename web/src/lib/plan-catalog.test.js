import { describe, it, expect } from 'vitest'
import { TIER_ORDER, getPlanName } from './plan-catalog'

describe('TIER_ORDER', () => {
  it('orders tiers from free to enterprise', () => {
    expect(TIER_ORDER.free).toBeLessThan(TIER_ORDER.pro)
    expect(TIER_ORDER.pro).toBeLessThan(TIER_ORDER.enterprise)
  })

  it('has exactly 3 tiers', () => {
    expect(Object.keys(TIER_ORDER)).toHaveLength(3)
  })
})

describe('getPlanName', () => {
  it('returns human-readable names for known slugs', () => {
    expect(getPlanName('free')).toBe('Free')
    expect(getPlanName('pro')).toBe('Pro')
    expect(getPlanName('enterprise')).toBe('Enterprise')
  })

  it('falls back to the raw slug for unknown plans', () => {
    expect(getPlanName('custom-plan')).toBe('custom-plan')
    expect(getPlanName('')).toBe('')
  })
})
