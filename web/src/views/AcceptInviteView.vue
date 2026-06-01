<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { UiButton, UiInput, UiField, UiCard, UiLoadingState, UiErrorState } from '@/components/ui'
import { CheckCircle, Lock, Sparkles } from 'lucide-vue-next'
import client from '@/api/client'

const route  = useRoute()
const router = useRouter()
const auth   = useAuthStore()

const token   = route.query.token || ''
const invite  = ref(null)
const loading = ref(true)
const error   = ref('')
const submitError = ref('')
const password   = ref('')
const confirm    = ref('')
const accepting  = ref(false)
const accepted   = ref(false)

// Peek invite info on load
onMounted(async () => {
  if (!token) { error.value = 'Missing invite token.'; loading.value = false; return }
  try {
    const { data } = await client.get('/invites/peek', { params: { token } })
    invite.value = data.invite
  } catch (e) {
    error.value = e.response?.data?.error?.message || 'Invite not found or has expired.'
  } finally {
    loading.value = false }
})

async function acceptInvite() {
  submitError.value = ''
  if (password.value !== confirm.value) { submitError.value = 'Passwords do not match.'; return }
  if (password.value.length < 8) { submitError.value = 'Password must be at least 8 characters.'; return }
  accepting.value = true
  try {
    const { data } = await client.post('/invites/accept', {
      token,
      password: password.value,
    })
    // Auto-login with the returned token
    auth.loginWithToken(data.access_token, data.user)
    accepted.value = true
    setTimeout(() => router.push('/'), 2000)
  } catch (e) {
    submitError.value = e.response?.data?.error?.message || 'Failed to accept invite. The link may have expired.'
  } finally {
    accepting.value = false }
}
</script>

<template>
  <div class="accept-page">
    <div class="accept-container animate-fadeIn">
      <!-- Brand -->
      <div class="accept-brand">
        <div class="brand-icon"><Sparkles :size="28" /></div>
        <h1 class="brand-title">You're invited</h1>
      </div>

      <!-- Loading -->
      <UiCard v-if="loading" padding="lg">
        <UiLoadingState mode="skeleton" :lines="3" />
      </UiCard>

      <!-- Error -->
      <UiCard v-else-if="error" padding="lg">
        <UiErrorState title="Invite unavailable" :message="error" :retry-label="''" />
        <div style="text-align:center;margin-top:1rem">
          <router-link to="/login" class="auth-link">Go to login →</router-link>
        </div>
      </UiCard>

      <!-- Accepted -->
      <UiCard v-else-if="accepted" padding="lg" class="accepted-card">
        <div class="accepted-inner">
          <CheckCircle :size="48" style="color:var(--color-primary)" />
          <h2>Welcome aboard!</h2>
          <p>Redirecting you to the dashboard…</p>
        </div>
      </UiCard>

      <!-- Accept form -->
      <template v-else-if="invite">
        <UiCard padding="lg">
          <div class="invite-info">
            <p class="invite-org">Join <strong>{{ invite.org_name }}</strong></p>
            <p class="invite-role">You've been invited as <span class="role-chip">{{ invite.role }}</span></p>
            <p class="invite-email">{{ invite.email }}</p>
          </div>

          <hr class="divider" />

          <p class="form-hint">Set a password to create your account and join the team.</p>

          <form @submit.prevent="acceptInvite">
            <div v-if="submitError" class="form-error" role="alert">{{ submitError }}</div>

            <UiField label="Password" html-for="accept-pw" :required="true">
              <UiInput
                id="accept-pw"
                v-model="password"
                type="password"
                placeholder="min. 8 characters"
                :icon="Lock"
                required
              />
            </UiField>

            <UiField label="Confirm password" html-for="accept-pw-confirm" :required="true">
              <UiInput
                id="accept-pw-confirm"
                v-model="confirm"
                type="password"
                placeholder="repeat password"
                :icon="Lock"
                required
              />
            </UiField>

            <UiButton type="submit" variant="primary" :loading="accepting" :full="true" size="lg">
              {{ accepting ? 'Joining…' : 'Accept invitation' }}
            </UiButton>
          </form>
        </UiCard>

        <div class="accept-footer">
          Already have an account? <router-link to="/login" class="auth-link">Sign in</router-link>
        </div>
      </template>
    </div>
    <div class="accept-bg" />
  </div>
</template>

<style scoped>
.accept-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  position: relative;
  overflow: hidden;
  padding: 2rem 1rem;
}
.accept-bg {
  position: absolute;
  top: -25%; left: -15%;
  width: 60vw; height: 60vw;
  border-radius: 50%;
  background: radial-gradient(circle, oklch(0.65 0.24 265 / 0.05), transparent 70%);
  pointer-events: none;
}
.accept-container { width: 100%; max-width: 440px; position: relative; z-index: 1; }
.accept-brand { text-align: center; margin-bottom: 2rem; }
.brand-icon {
  display: inline-flex; align-items: center; justify-content: center;
  width: 56px; height: 56px; border-radius: var(--radius-lg);
  background: var(--color-primary); color: var(--color-text-on-primary);
  margin-bottom: 1rem; box-shadow: var(--shadow-glow);
}
.brand-title { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; }

.invite-info { text-align: center; }
.invite-org { font-size: 1.25rem; font-weight: 600; margin-bottom: 0.5rem; }
.invite-role { font-size: 0.9375rem; color: var(--color-text-secondary); margin-bottom: 0.25rem; }
.invite-email { font-size: 0.875rem; color: var(--color-text-tertiary); }
.role-chip {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--color-primary-subtle);
  color: var(--color-primary);
  font-weight: 600;
  font-size: 0.8125rem;
  text-transform: capitalize;
}
.divider { border: none; border-top: 1px solid var(--color-border-subtle); margin: 1.25rem 0; }
.form-hint { font-size: 0.875rem; color: var(--color-text-secondary); margin-bottom: 1.25rem; }
.form-error {
  padding: 0.75rem 1rem;
  background: oklch(0.95 0.05 25);
  color: var(--color-danger);
  border-radius: var(--radius-md);
  font-size: 0.8125rem;
  margin-bottom: 1.25rem;
}
@media (prefers-color-scheme: dark) { .form-error { background: oklch(0.22 0.05 25); } }

.accepted-card { text-align: center; }
.accepted-inner { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 1rem 0; }
.accepted-inner h2 { font-size: 1.25rem; font-weight: 700; }
.accepted-inner p { color: var(--color-text-secondary); font-size: 0.875rem; }

.accept-footer { text-align: center; margin-top: 1.5rem; font-size: 0.875rem; color: var(--color-text-secondary); }
.auth-link { color: var(--color-primary); font-weight: 500; }
</style>
