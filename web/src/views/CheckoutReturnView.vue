<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useBillingStore } from '@/stores/billing'
import { UiButton, UiLoadingState } from '@/components/ui'
import { CheckCircle, XCircle, ArrowRight, RefreshCw } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const billing = useBillingStore()

// Determine success vs cancel from route path
const isSuccess = computed(() => route.path.endsWith('/success'))

// Poll for plan activation after successful checkout
const pollingActive = ref(false)
const pollCount = ref(0)
const MAX_POLLS = 8
const POLL_INTERVAL_MS = 2500

onMounted(async () => {
  if (isSuccess.value) {
    pollingActive.value = true
    poll()
  }
})

function poll() {
  if (pollCount.value >= MAX_POLLS) {
    pollingActive.value = false
    return
  }
  setTimeout(async () => {
    pollCount.value++
    await billing.fetchPlan(true)
    // If plan is now upgraded (not free), stop polling
    if (billing.planSlug && billing.planSlug !== 'free') {
      pollingActive.value = false
    } else {
      poll()
    }
  }, POLL_INTERVAL_MS)
}

function goToBilling() {
  router.push('/settings/billing')
}
</script>

<template>
  <div class="checkout-return-page animate-fadeIn">
    <div class="return-card">

      <!-- Success state -->
      <template v-if="isSuccess">
        <!-- While polling for plan activation -->
        <template v-if="pollingActive">
          <div class="return-icon return-icon--loading">
            <RefreshCw :size="36" class="spin" style="color:var(--color-primary)" />
          </div>
          <h1 class="return-title">Activating your plan…</h1>
          <p class="return-message">
            Your payment was successful. We're confirming your upgraded plan — this usually takes just a few seconds.
          </p>
          <UiLoadingState mode="pulse" />
        </template>

        <!-- Plan confirmed upgraded -->
        <template v-else-if="billing.planSlug && billing.planSlug !== 'free'">
          <div class="return-icon return-icon--success">
            <CheckCircle :size="40" style="color:var(--color-success)" />
          </div>
          <h1 class="return-title">You're on {{ billing.planSlug.charAt(0).toUpperCase() + billing.planSlug.slice(1) }}!</h1>
          <p class="return-message">
            Your plan has been upgraded. Your new limits and features are now active.
          </p>
          <UiButton variant="primary" @click="goToBilling">
            <ArrowRight :size="15" /> View Plan &amp; Usage
          </UiButton>
        </template>

        <!-- Poll timed out — payment received but webhook may be delayed -->
        <template v-else>
          <div class="return-icon return-icon--success">
            <CheckCircle :size="40" style="color:var(--color-success)" />
          </div>
          <h1 class="return-title">Upgrade Processing</h1>
          <p class="return-message">
            Your plan upgrade is being activated. This typically takes a few seconds.
          </p>
          <p class="return-hint">
            If your plan hasn't updated within a few minutes, please contact support.
          </p>
          <UiButton variant="secondary" @click="goToBilling">
            Back to Billing
          </UiButton>
        </template>
      </template>

      <!-- Cancelled state -->
      <template v-else>
        <div class="return-icon return-icon--cancel">
          <XCircle :size="40" style="color:var(--color-text-secondary)" />
        </div>
        <h1 class="return-title">Checkout Cancelled</h1>
        <p class="return-message">
          No changes have been made to your plan or billing.
        </p>
        <UiButton variant="secondary" @click="goToBilling">
          Back to Billing
        </UiButton>
      </template>

    </div>
  </div>
</template>

<style scoped>
.checkout-return-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 70vh;
  padding: 2rem;
}

.return-card {
  max-width: 480px;
  width: 100%;
  text-align: center;
  padding: 3rem 2rem;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.return-icon { margin-bottom: 0.5rem; }

.return-title {
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: -0.025em;
}

.return-message {
  font-size: 0.9375rem;
  color: var(--color-text-secondary);
  line-height: 1.6;
  max-width: 360px;
}

.return-hint {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  max-width: 360px;
}

/* Spinning icon for loading state */
.spin {
  animation: spin 1.2s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
