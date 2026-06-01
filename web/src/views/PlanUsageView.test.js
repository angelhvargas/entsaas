import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PlanUsageView from './PlanUsageView.vue'
import { useBillingStore } from '@/stores/billing'
import { useRouter } from 'vue-router'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

describe('PlanUsageView.vue Component', () => {
  let pinia

  beforeEach(() => {
    localStorage.clear() // Prevent auth store auto-profile hydration from executing
    vi.restoreAllMocks()
    mockPush.mockReset()
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('renders loading state when store is loading', () => {
    const store = useBillingStore()
    store.loading = true
    store._fetched = true // Prevent fetchPlan from firing real network requests during mount

    const wrapper = mount(PlanUsageView, {
      global: {
        plugins: [pinia],
        mocks: {
          $router: {
            push: mockPush,
          },
        },
        stubs: {
          UiCard: true,
          UiButton: true,
          UiBadge: true,
          UiLoadingState: true,
          UiEmptyState: true,
        },
      },
    })

    // Assert loading state is displayed
    expect(wrapper.findComponent({ name: 'UiLoadingState' }).exists()).toBe(true)
  })

  it('renders plan comparisons and entitlements correctly when loaded', async () => {
    const store = useBillingStore()
    
    // Set up mock loaded state
    store._fetched = true
    store.planSlug = 'free'
    store.entitlements = {
      max_projects: 3,
      max_members: 3,
      ai_assistant_enabled: false,
      priority_support: false,
    }
    
    // Structure catalogPlans exactly matching the dynamic schema mapper requirements
    store.catalogPlans = [
      {
        id: 'free',
        name: 'Free Tier',
        price_monthly: 0,
        capacity: { max_projects: 3, max_members: 3 },
        features: { ai_assistant_enabled: false },
      },
      {
        id: 'pro',
        name: 'Pro Tier',
        price_monthly: 49,
        capacity: { max_projects: -1, max_members: -1 },
        features: { ai_assistant_enabled: true },
      },
    ]

    const wrapper = mount(PlanUsageView, {
      global: {
        plugins: [pinia],
        mocks: {
          $router: {
            push: mockPush,
          },
        },
        stubs: {
          UiCard: { template: '<div><slot /><slot name="footer" /></div>' },
          UiButton: { template: '<button><slot /></button>' },
          UiBadge: { template: '<span><slot /></span>' },
          UiLoadingState: true,
          UiEmptyState: true,
        },
      },
    })

    // Assert that active plans are displayed
    expect(wrapper.text()).toContain('Free Tier')
    expect(wrapper.text()).toContain('Pro Tier')

    // Assert dynamic feature list maps properly
    expect(wrapper.text()).toContain('Up to 3 projects')
    expect(wrapper.text()).toContain('Unlimited projects')
    expect(wrapper.text()).toContain('AI assistant')

    // Assert that the upgrade button for Pro is rendered
    const buttons = wrapper.findAll('button')
    const upgradeButton = buttons.find(b => b.text().includes('Upgrade to Pro'))
    expect(upgradeButton).toBeDefined()
  })

  it('triggers router push when upgrading to enterprise tier', async () => {
    const store = useBillingStore()
    store._fetched = true
    store.planSlug = 'pro' // Next plan will be enterprise
    store.entitlements = {
      max_projects: 10,
    }
    
    store.catalogPlans = [
      {
        id: 'free',
        name: 'Free Tier',
        capacity: { max_projects: 3 },
      },
      {
        id: 'pro',
        name: 'Pro Tier',
        capacity: { max_projects: -1 },
      },
      {
        id: 'enterprise',
        name: 'Enterprise',
        capacity: { max_projects: -1 },
      },
    ]

    const wrapper = mount(PlanUsageView, {
      global: {
        plugins: [pinia],
        mocks: {
          $router: {
            push: mockPush,
          },
        },
        stubs: {
          UiCard: { template: '<div><slot /><slot name="footer" /></div>' },
          UiButton: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
          UiBadge: true,
          UiLoadingState: true,
          UiEmptyState: true,
        },
      },
    })

    const buttons = wrapper.findAll('button')
    const contactBtn = buttons.find(b => b.text().includes('Contact us'))
    expect(contactBtn).toBeDefined()

    // Trigger upgrade to Enterprise
    await contactBtn.trigger('click')

    // Pro -> Enterprise is a redirect to settings
    expect(mockPush).toHaveBeenCalledWith('/settings')
  })
})
