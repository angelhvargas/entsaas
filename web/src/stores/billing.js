import { defineStore } from 'pinia'
import client from '@/api/client'

export const useBillingStore = defineStore('billing', {
  state: () => ({
    planSlug: null,       // 'free', 'pro', 'enterprise'
    entitlements: {},     // { max_projects: 3, max_members: 3, ai_assistant_enabled: false, ... }
    catalogPlans: [],     // Dynamically loaded plans from configuration
    loading: false,
    error: null,
    _fetched: false,      // prevents duplicate plan fetches
  }),

  getters: {
    /** @returns {boolean} Whether plan data has been loaded */
    hasPlan: (state) => !!state.planSlug,

    /**
     * Check if a feature flag is enabled on the current plan.
     * Returns true if plan is not loaded (fail-open for UX during load).
     */
    canDo: (state) => (feature) => {
      if (!state._fetched) return true // fail-open until loaded
      return !!state.entitlements[feature]
    },

    /**
     * Check if a resource limit has been reached.
     * Returns false if plan is not loaded (fail-open).
     */
    isAtLimit: (state) => (resource, current) => {
      if (!state._fetched) return false
      const limit = state.entitlements[resource]
      if (limit === undefined || limit === -1) return false // unlimited
      return current >= limit
    },
  },

  actions: {
    /**
     * Fetch the org's effective plan and entitlements.
     * Idempotent — only fetches once unless force=true.
     */
    async fetchPlan(force = false) {
      if (this._fetched && !force) return
      this.loading = true
      this.error = null
      try {
        const [planRes, catalogRes] = await Promise.all([
          client.get('/billing/plan'),
          client.get('/billing/plans')
        ])
        this.planSlug = planRes.data.plan_slug || 'free'
        this.entitlements = planRes.data.entitlements || {}
        this.catalogPlans = catalogRes.data.plans || []
        this._fetched = true
      } catch (err) {
        this.error = err.message || 'Failed to load plan entitlements'
      } finally {
        this.loading = false
      }
    },

    /** Reset store state (e.g., on logout). */
    $reset() {
      this.planSlug = null
      this.entitlements = {}
      this.loading = false
      this.error = null
      this._fetched = false
    },
  },
})
