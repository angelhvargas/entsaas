<script setup>
import { ref, onMounted, computed } from 'vue'
import { CreditCard, Zap, Building2, Check, ExternalLink, AlertCircle } from 'lucide-vue-next'
import { UiCard, UiButton, UiLoadingState, UiErrorState, UiBadge, UiTabs } from '@/components/ui'
import client from '@/api/client'
import { useRoute } from 'vue-router'

const route = useRoute()
const plans       = ref([])
const subscription = ref(null)
const loading     = ref(true)
const error       = ref('')
const checkoutLoading = ref('')
const portalLoading   = ref(false)

const activeTab = ref('overview')
const tabs = [
  { key: 'overview', label: 'Overview',  icon: CreditCard },
  { key: 'plans',    label: 'Plans',     icon: Zap },
]

// Toast from checkout redirect
const checkoutResult = route.query.checkout || ''

onMounted(async () => {
  try {
    const [plansRes, subRes] = await Promise.all([
      client.get('/billing/plans'),
      client.get('/billing/subscription'),
    ])
    plans.value = plansRes.data.plans || []
    subscription.value = subRes.data.subscription
  } catch (e) {
    error.value = 'Failed to load billing information.'
  } finally {
    loading.value = false }
})

const currentPlan = computed(() =>
  plans.value.find(p => p.id === subscription.value?.plan_id)
)

const statusVariant = computed(() => {
  const s = subscription.value?.status
  return s === 'active' ? 'success' : s === 'trialing' ? 'primary' : s === 'past_due' ? 'danger' : 'default'
})

async function upgradeTo(planId) {
  checkoutLoading.value = planId
  try {
    const { data } = await client.post('/billing/checkout', { plan_id: planId })
    if (data.checkout_url && data.checkout_url !== '#') {
      window.location.href = data.checkout_url
    }
  } catch (e) {
    error.value = 'Failed to start checkout. Please try again.'
  } finally {
    checkoutLoading.value = '' }
}

async function openPortal() {
  portalLoading.value = true
  try {
    const { data } = await client.post('/billing/portal')
    if (data.portal_url && data.portal_url !== '#') {
      window.open(data.portal_url, '_blank')
    }
  } catch {} finally { portalLoading.value = false }
}

function formatPrice(price) {
  if (!price) return 'Free'
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'usd', maximumFractionDigits: 0 }).format(price)
}

const planIcons = { free: CreditCard, pro: Zap, enterprise: Building2 }
</script>

<template>
  <div class="animate-fadeIn">
    <div class="page-header">
      <div>
        <h1 class="page-title">Billing</h1>
        <p class="page-subtitle">Manage your subscription and plan</p>
      </div>
    </div>

    <!-- Checkout result banner -->
    <div v-if="checkoutResult === 'success'" class="banner banner--success">
      <Check :size="16" /> Payment successful — your plan has been updated.
    </div>
    <div v-else-if="checkoutResult === 'cancelled'" class="banner banner--info">
      <AlertCircle :size="16" /> Checkout cancelled.
    </div>

    <UiLoadingState v-if="loading" mode="skeleton" :lines="4" />
    <UiErrorState v-else-if="error" :message="error" @retry="onMounted" />

    <template v-else>
      <UiTabs v-model="activeTab" :tabs="tabs">
        <!-- Overview tab -->
        <div v-if="activeTab === 'overview'" class="overview">
          <UiCard padding="md">
            <template #header>
              <div class="card-header-row"><CreditCard :size="18" style="color:var(--color-primary)" /> Current Subscription</div>
            </template>
            <div class="sub-info">
              <div class="sub-row">
                <span class="sub-label">Plan</span>
                <span class="sub-value plan-name">{{ currentPlan?.name || '—' }}</span>
              </div>
              <div class="sub-row">
                <span class="sub-label">Status</span>
                <UiBadge :variant="statusVariant" dot size="sm" style="text-transform:capitalize">
                  {{ subscription?.status || '—' }}
                </UiBadge>
              </div>
              <div class="sub-row" v-if="subscription?.current_period_end">
                <span class="sub-label">Renews</span>
                <span class="sub-value">{{ new Date(subscription.current_period_end).toLocaleDateString() }}</span>
              </div>
              <div v-if="subscription?.cancel_at_period_end" class="cancel-notice">
                <AlertCircle :size="14" /> Your subscription will be cancelled at the end of the current period.
              </div>
            </div>
            <template #footer>
              <UiButton variant="secondary" size="sm" :loading="portalLoading" @click="openPortal">
                <ExternalLink :size="14" /> Manage in billing portal
              </UiButton>
            </template>
          </UiCard>
        </div>

        <!-- Plans tab -->
        <div v-else-if="activeTab === 'plans'" class="plans-grid">
          <UiCard
            v-for="plan in plans"
            :key="plan.id"
            padding="lg"
            class="plan-card"
            :class="{ 'plan-card--current': plan.id === subscription?.plan_id }"
          >
            <div class="plan-icon-wrap">
              <component :is="planIcons[plan.id] || Zap" :size="24" style="color:var(--color-primary)" />
            </div>
            <h3 class="plan-name">{{ plan.name }}</h3>
            <div class="plan-price">
              <span class="plan-amount">{{ formatPrice(plan.price_monthly) }}</span>
              <span v-if="plan.price_monthly" class="plan-period">/mo</span>
            </div>
            <p class="plan-desc">{{ plan.description }}</p>

            <ul v-if="plan.features?.length" class="plan-features">
              <li v-for="f in plan.features" :key="f">
                <Check :size="13" style="color:var(--color-primary);flex-shrink:0" /> {{ f }}
              </li>
            </ul>

            <div class="plan-action">
              <UiBadge v-if="plan.id === subscription?.plan_id" variant="success" size="sm" dot>Current plan</UiBadge>
              <UiButton
                v-else-if="plan.id === 'enterprise'"
                variant="secondary"
                :full="true"
                @click="$router.push('/settings')"
              >
                Contact us
              </UiButton>
              <UiButton
                v-else
                variant="primary"
                :full="true"
                :loading="checkoutLoading === plan.id"
                @click="upgradeTo(plan.id)"
              >
                Upgrade to {{ plan.name }}
              </UiButton>
            </div>
          </UiCard>
        </div>
      </UiTabs>
    </template>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 2rem; }
.page-title { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.page-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; margin-top: 0.25rem; }

.banner {
  display: flex; align-items: center; gap: 8px;
  padding: 0.75rem 1rem; border-radius: var(--radius-md); margin-bottom: 1.5rem;
  font-size: 0.875rem;
}
.banner--success { background: oklch(0.92 0.06 155); color: oklch(0.45 0.15 155); }
.banner--info { background: var(--color-bg-muted); color: var(--color-text-secondary); }
@media (prefers-color-scheme: dark) {
  .banner--success { background: oklch(0.22 0.05 155); color: oklch(0.72 0.19 155); }
}

.card-header-row { display: flex; align-items: center; gap: 8px; }

.sub-info { display: flex; flex-direction: column; }
.sub-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 0; border-bottom: 1px solid var(--color-border-subtle);
}
.sub-row:last-child { border-bottom: none; }
.sub-label { font-size: 0.875rem; color: var(--color-text-secondary); }
.sub-value { font-size: 0.875rem; font-weight: 500; }
.plan-name { font-weight: 700; }
.cancel-notice {
  display: flex; align-items: center; gap: 6px;
  padding: 0.5rem 0; font-size: 0.8125rem; color: var(--color-warning);
}

.plans-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
.plan-card { display: flex; flex-direction: column; }
.plan-card--current { border-color: var(--color-primary); }

.plan-icon-wrap {
  width: 44px; height: 44px; border-radius: var(--radius-md);
  background: var(--color-primary-subtle);
  display: flex; align-items: center; justify-content: center;
  margin-bottom: 1rem;
}
.plan-name { font-size: 1.125rem; font-weight: 700; margin-bottom: 0.25rem; }
.plan-price { display: flex; align-items: baseline; gap: 2px; margin-bottom: 0.5rem; }
.plan-amount { font-size: 2rem; font-weight: 800; letter-spacing: -0.03em; }
.plan-period { font-size: 0.875rem; color: var(--color-text-secondary); }
.plan-desc { font-size: 0.875rem; color: var(--color-text-secondary); margin-bottom: 1rem; }
.plan-features { list-style: none; padding: 0; margin: 0 0 1.5rem; display: flex; flex-direction: column; gap: 8px; flex: 1; }
.plan-features li { display: flex; align-items: center; gap: 6px; font-size: 0.875rem; }
.plan-action { margin-top: auto; }
</style>
