import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useBillingStore } from './billing'
import { server } from '../test/server'
import { http, HttpResponse } from 'msw'

describe('billing.js Pinia Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('has correct default state', () => {
    const store = useBillingStore()
    expect(store.planSlug).toBeNull()
    expect(store.entitlements).toEqual({})
    expect(store.catalogPlans).toEqual([])
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
    expect(store._fetched).toBe(false)
  })

  describe('Getters', () => {
    it('hasPlan checks if planSlug is set', () => {
      const store = useBillingStore()
      expect(store.hasPlan).toBe(false)
      store.planSlug = 'pro'
      expect(store.hasPlan).toBe(true)
    })

    it('canDo fails open when not yet fetched, then returns correct entitlements', () => {
      const store = useBillingStore()
      // 1. Fail-open behavior (unfetched)
      expect(store.canDo('ai_assistant_enabled')).toBe(true)

      // 2. Mock fetched state
      store._fetched = true
      store.entitlements = {
        ai_assistant_enabled: false,
        priority_support: true,
      }

      expect(store.canDo('ai_assistant_enabled')).toBe(false)
      expect(store.canDo('priority_support')).toBe(true)
      expect(store.canDo('non_existent')).toBe(false)
    })

    it('isAtLimit handles limits and unlimited resources correctly', () => {
      const store = useBillingStore()
      // 1. Fail-open behavior (unfetched)
      expect(store.isAtLimit('max_projects', 10)).toBe(false)

      // 2. Fetch and test limits
      store._fetched = true
      store.entitlements = {
        max_projects: 3,
        max_members: -1, // unlimited
      }

      // Within limit
      expect(store.isAtLimit('max_projects', 1)).toBe(false)
      // At limit
      expect(store.isAtLimit('max_projects', 3)).toBe(true)
      // Over limit
      expect(store.isAtLimit('max_projects', 5)).toBe(true)

      // Unlimited checks
      expect(store.isAtLimit('max_members', 0)).toBe(false)
      expect(store.isAtLimit('max_members', 9999)).toBe(false)
      expect(store.isAtLimit('non_existent_resource', 5)).toBe(false)
    })
  })

  describe('Actions', () => {
    it('fetchPlan fetches effective plan and catalog plans successfully', async () => {
      const store = useBillingStore()

      server.use(
        http.get('/v1/billing/plan', () => {
          return HttpResponse.json({
            plan_slug: 'pro',
            entitlements: { max_projects: 10, ai_enabled: true },
          })
        }),
        http.get('/v1/billing/plans', () => {
          return HttpResponse.json({
            plans: [
              { id: 'free', name: 'Free' },
              { id: 'pro', name: 'Pro' },
            ],
          })
        })
      )

      await store.fetchPlan()

      expect(store.planSlug).toBe('pro')
      expect(store.entitlements).toEqual({ max_projects: 10, ai_enabled: true })
      expect(store.catalogPlans).toHaveLength(2)
      expect(store._fetched).toBe(true)
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })

    it('fetchPlan handles errors gracefully', async () => {
      const store = useBillingStore()

      server.use(
        http.get('/v1/billing/plan', () => {
          return new HttpResponse(null, { status: 500, statusText: 'Server Error' })
        })
      )

      await store.fetchPlan()

      expect(store.planSlug).toBeNull()
      expect(store.loading).toBe(false)
      expect(store._fetched).toBe(false)
      expect(store.error).not.toBeNull()
    })

    it('$reset resets the store completely', () => {
      const store = useBillingStore()
      store.planSlug = 'enterprise'
      store.entitlements = { unlimited: true }
      store.loading = true
      store.error = 'some error'
      store._fetched = true

      store.$reset()

      expect(store.planSlug).toBeNull()
      expect(store.entitlements).toEqual({})
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store._fetched).toBe(false)
    })
  })
})
