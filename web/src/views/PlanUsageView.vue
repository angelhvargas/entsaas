<script setup>
import { computed, onMounted } from 'vue'
import { useBillingStore } from '@/stores/billing'
import {
  UiCard, UiButton, UiBadge, UiLoadingState, UiEmptyState,
} from '@/components/ui'
import {
  CreditCard, AlertTriangle, CheckCircle, XCircle, ArrowUpCircle, Zap, Building2,
} from 'lucide-vue-next'
import { usagePercent, formatUsage, usageStatusLabel } from '@/lib/usage'
import client from '@/api/client'
import { useRouter } from 'vue-router'
import { ref } from 'vue'

const billing = useBillingStore()
const router = useRouter()
const checkoutLoading = ref(false)
const checkoutError = ref('')

const planVariant = computed(() => {
  const map = { free: 'default', pro: 'primary', enterprise: 'info' }
  return map[billing.planSlug] ?? 'default'
})

const canUpgrade = computed(() => {
  return billing.planSlug === 'free' || billing.planSlug === 'pro'
})

const nextPlanSlug = computed(() => {
  const next = { free: 'pro', pro: 'enterprise' }
  return next[billing.planSlug] ?? null
})

// Feature entitlements derived from the billing store
const featureEntitlements = computed(() => {
  const featureKeys = [
    'ai_assistant_enabled',
    'sso_enabled',
    'priority_support',
  ]
  return featureKeys.map(key => ({
    key,
    label: featureLabel(key),
    enabled: !!billing.entitlements[key],
  }))
})

// Capacity entitlements derived from the billing store
const capacityEntitlements = computed(() => {
  const capacityKeys = [
    { key: 'max_projects', label: 'Projects' },
    { key: 'max_members', label: 'Team members' },
    { key: 'audit_log_retention_days', label: 'Audit log retention (days)' },
  ]
  return capacityKeys
    .filter(c => billing.entitlements[c.key] !== undefined)
    .map(c => ({
      ...c,
      limit: billing.entitlements[c.key],
    }))
})

function featureLabel(key) {
  const labels = {
    ai_assistant_enabled: 'AI Assistant',
    sso_enabled: 'SSO / SAML',
    priority_support: 'Priority Support',
  }
  return labels[key] ?? key
}

function limitLabel(limit) {
  if (limit === -1 || limit < 0) return 'Unlimited'
  return limit
}

function barClass(pct) {
  if (pct >= 100) return 'fill--error'
  if (pct >= 80) return 'fill--warning'
  return 'fill--ok'
}

async function handleUpgrade() {
  const target = nextPlanSlug.value
  if (!target) return
  if (target === 'enterprise') {
    router.push('/settings')
    return
  }
  checkoutLoading.value = true
  checkoutError.value = ''
  try {
    const { data } = await client.post('/billing/checkout', { plan_id: target })
    if (data.checkout_url && data.checkout_url !== '#') {
      window.location.href = data.checkout_url
    }
  } catch (e) {
    checkoutError.value = 'Failed to start checkout. Please try again.'
  } finally {
    checkoutLoading.value = false
  }
}

const PLAN_COMPARISON = computed(() => {
  return billing.catalogPlans.map(plan => {
    // Map icons and features based on configuration
    let icon = CreditCard
    if (plan.id === 'pro') icon = Zap
    if (plan.id === 'enterprise') icon = Building2

    const features = []
    if (plan.capacity?.max_projects === -1) features.push('Unlimited projects')
    else if (plan.capacity?.max_projects > 0) features.push(`Up to ${plan.capacity.max_projects} projects`)
    
    if (plan.capacity?.max_members === -1) features.push('Unlimited team members')
    else if (plan.capacity?.max_members > 0) features.push(`Up to ${plan.capacity.max_members} team members`)

    if (plan.features?.ai_assistant_enabled) features.push('AI assistant')
    if (plan.features?.sso_enabled) features.push('SSO / SAML')
    if (plan.features?.priority_support) features.push('Priority support')
    else features.push('Community support')

    if (plan.capacity?.audit_log_retention_days === -1) features.push('Unlimited audit log')
    else if (plan.capacity?.audit_log_retention_days > 0) features.push(`${plan.capacity.audit_log_retention_days}-day audit log`)

    return {
      id: plan.id,
      name: plan.name,
      price: plan.price_monthly === 0 ? '$0' : (plan.price_monthly ? `$${plan.price_monthly}` : 'Custom'),
      period: plan.price_monthly > 0 ? '/ month' : (plan.price_monthly === 0 ? '/ forever' : 'pricing'),
      icon: icon,
      selfServe: plan.is_self_serve,
      highlighted: plan.id === 'pro',
      features: features,
    }
  })
})

onMounted(() => billing.fetchPlan())
</script>

<template>
  <div class="animate-fadeIn">
    <!-- Page header -->
    <div class="page-header">
      <div class="page-header-icon">
        <CreditCard :size="18" style="color:var(--color-primary)" />
      </div>
      <div>
        <h1 class="page-title">Plan &amp; Usage</h1>
        <p class="page-subtitle">Your current plan, entitlements, and resource limits.</p>
      </div>
    </div>

    <UiLoadingState v-if="billing.loading && !billing.hasPlan" mode="skeleton" :lines="4" />

    <template v-else>

      <!-- Error banner -->
      <div v-if="checkoutError" class="banner banner--error" style="margin-bottom:1.5rem">
        <AlertTriangle :size="15" /> {{ checkoutError }}
      </div>

      <!-- Current plan card -->
      <UiCard padding="md" style="margin-bottom:1.5rem">
        <template #header>
          <div class="card-header-row">
            <CreditCard :size="16" style="color:var(--color-primary)" />
            Current Plan
          </div>
        </template>

        <div class="plan-summary-rows">
          <div class="plan-row">
            <span class="plan-label">Plan</span>
            <UiBadge :variant="planVariant" dot>
              {{ billing.planSlug ?? '—' }}
            </UiBadge>
          </div>
          <div class="plan-row" v-for="cap in capacityEntitlements" :key="cap.key">
            <span class="plan-label">{{ cap.label }}</span>
            <span class="plan-value">{{ limitLabel(cap.limit) }}</span>
          </div>
        </div>

        <template #footer v-if="canUpgrade">
          <UiButton variant="primary" size="sm" :loading="checkoutLoading" @click="handleUpgrade">
            <Zap :size="14" /> Upgrade to {{ nextPlanSlug === 'pro' ? 'Pro' : 'Enterprise' }}
          </UiButton>
          <span class="upgrade-hint">Self-serve upgrade via secure checkout</span>
        </template>
      </UiCard>

      <!-- Features included -->
      <UiCard padding="md" style="margin-bottom:1.5rem">
        <template #header>
          <div class="card-header-row">
            <CheckCircle :size="16" style="color:var(--color-primary)" />
            Features Included
          </div>
        </template>
        <div class="features-list">
          <div
            v-for="feat in featureEntitlements"
            :key="feat.key"
            class="feature-row"
          >
            <component
              :is="feat.enabled ? CheckCircle : XCircle"
              :size="15"
              :style="{ color: feat.enabled ? 'var(--color-success)' : 'var(--color-text-secondary)' }"
            />
            <span :class="{ 'feature-disabled': !feat.enabled }">{{ feat.label }}</span>
          </div>
        </div>
        <UiEmptyState v-if="!featureEntitlements.length" message="No feature data" />
      </UiCard>

      <!-- Plan comparison -->
      <UiCard padding="md" style="margin-bottom:1.5rem">
        <template #header>
          <div class="card-header-row">
            <ArrowUpCircle :size="16" style="color:var(--color-primary)" />
            Plan Comparison
          </div>
        </template>
        <p class="section-desc">See what's included at each tier. Your current plan is highlighted.</p>
        <div class="plan-compare-grid">
          <div
            v-for="col in PLAN_COMPARISON"
            :key="col.id"
            class="plan-col"
            :class="{
              'plan-col--current': billing.planSlug === col.id,
              'plan-col--highlighted': col.highlighted,
            }"
          >
            <div class="plan-col-header">
              <div class="plan-col-icon">
                <component :is="col.icon" :size="18" style="color:var(--color-primary)" />
              </div>
              <UiBadge v-if="billing.planSlug === col.id" variant="primary" size="sm" dot>Current</UiBadge>
            </div>
            <div class="plan-col-name">{{ col.name }}</div>
            <div class="plan-col-price">
              {{ col.price }}<span class="plan-col-period">{{ col.period }}</span>
            </div>
            <ul class="plan-col-features">
              <li v-for="f in col.features" :key="f">
                <CheckCircle :size="12" style="color:var(--color-success);flex-shrink:0" />
                {{ f }}
              </li>
            </ul>
            <div class="plan-col-action">
              <span v-if="billing.planSlug === col.id" class="plan-current-label">Your plan</span>
              <UiButton
                v-else-if="col.selfServe && canUpgrade && nextPlanSlug === col.id"
                variant="primary"
                :full="true"
                size="sm"
                :loading="checkoutLoading"
                @click="handleUpgrade"
              >
                Upgrade to {{ col.name }}
              </UiButton>
              <UiButton
                v-else-if="col.id === 'enterprise'"
                variant="secondary"
                :full="true"
                size="sm"
                @click="$router.push('/settings')"
              >
                Contact us
              </UiButton>
            </div>
          </div>
        </div>
      </UiCard>

    </template>
  </div>
</template>

<style scoped>
.page-header {
  display: flex; align-items: flex-start; gap: 14px; margin-bottom: 2rem;
}
.page-header-icon {
  width: 38px; height: 38px; border-radius: var(--radius-md);
  background: var(--color-primary-subtle);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.page-title { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.page-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; margin-top: 0.25rem; }

.card-header-row { display: flex; align-items: center; gap: 8px; }

.banner {
  display: flex; align-items: center; gap: 8px;
  padding: 0.75rem 1rem; border-radius: var(--radius-md);
  font-size: 0.875rem;
}
.banner--error {
  background: oklch(0.92 0.06 25); color: oklch(0.45 0.2 25);
  border: 1px solid oklch(0.82 0.1 25);
}
@media (prefers-color-scheme: dark) {
  .banner--error { background: oklch(0.22 0.06 25); color: oklch(0.72 0.2 25); border-color: oklch(0.35 0.08 25); }
}

/* Plan summary */
.plan-summary-rows { display: flex; flex-direction: column; }
.plan-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 0; border-bottom: 1px solid var(--color-border-subtle);
  font-size: 0.875rem;
}
.plan-row:last-child { border-bottom: none; }
.plan-label { color: var(--color-text-secondary); }
.plan-value { font-weight: 600; }

.upgrade-hint {
  margin-left: 0.75rem;
  font-size: 0.75rem; color: var(--color-text-secondary);
}

/* Features */
.section-desc { font-size: 0.8125rem; color: var(--color-text-secondary); margin-bottom: 1rem; }
.features-list { display: flex; flex-direction: column; gap: 10px; }
.feature-row { display: flex; align-items: center; gap: 10px; font-size: 0.875rem; }
.feature-disabled { color: var(--color-text-secondary); text-decoration: line-through; }

/* Plan comparison grid */
.plan-compare-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}
@media (max-width: 768px) {
  .plan-compare-grid { grid-template-columns: 1fr; }
}

.plan-col {
  background: var(--color-bg-muted);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex; flex-direction: column; gap: 0.75rem;
  transition: border-color var(--transition-fast);
}
.plan-col--current {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary);
}
.plan-col--highlighted {
  background: var(--color-primary-subtle);
  border-color: var(--color-primary);
}

.plan-col-header { display: flex; align-items: center; justify-content: space-between; }
.plan-col-icon {
  width: 34px; height: 34px; border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  display: flex; align-items: center; justify-content: center;
}
.plan-col-name { font-size: 1.0625rem; font-weight: 700; }
.plan-col-price { font-size: 1.5rem; font-weight: 800; letter-spacing: -0.03em; }
.plan-col-period { font-size: 0.8125rem; font-weight: 400; color: var(--color-text-secondary); margin-left: 3px; }

.plan-col-features {
  list-style: none; padding: 0; margin: 0;
  display: flex; flex-direction: column; gap: 7px; flex: 1;
}
.plan-col-features li {
  display: flex; align-items: center; gap: 6px;
  font-size: 0.8125rem; color: var(--color-text-secondary);
}

.plan-col-action { margin-top: auto; padding-top: 0.75rem; }
.plan-current-label { font-size: 0.8125rem; font-weight: 600; color: var(--color-primary); }
</style>
