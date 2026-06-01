<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { UiButton, UiInput, UiField } from '@/components/ui'
import { LogIn, ArrowRight, Sparkles, Mail, Lock } from 'lucide-vue-next'

const router = useRouter()
const route  = useRoute()
const auth   = useAuthStore()

const email    = ref('')
const password = ref('')
const error    = ref('')
const loading  = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value)
    router.push(route.query.redirect || '/')
  } catch (err) {
    error.value = err.response?.data?.error?.message || 'Login failed. Please try again.'
  } finally {
    loading.value = false }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-container animate-fadeIn">
      <!-- Brand -->
      <div class="auth-brand">
        <div class="brand-icon"><Sparkles :size="28" /></div>
        <h1 class="brand-title">EntSaaS</h1>
        <p class="brand-subtitle">Sign in to your account</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="handleLogin" class="auth-form">
        <div v-if="error" class="auth-error" role="alert">{{ error }}</div>

        <UiField label="Email" html-for="login-email" :required="true">
          <UiInput
            id="login-email"
            v-model="email"
            type="email"
            placeholder="you@company.com"
            :icon="Mail"
            required
            autocomplete="email"
          />
        </UiField>

        <UiField label="Password" html-for="login-password" :required="true">
          <UiInput
            id="login-password"
            v-model="password"
            type="password"
            placeholder="••••••••"
            :icon="Lock"
            required
            autocomplete="current-password"
          />
        </UiField>

        <div class="auth-actions">
          <router-link to="/forgot-password" class="auth-link">Forgot password?</router-link>
        </div>

        <UiButton type="submit" variant="primary" :loading="loading" :full="true" size="lg">
          <LogIn :size="16" />
          {{ loading ? 'Signing in…' : 'Sign in' }}
          <ArrowRight v-if="!loading" :size="16" />
        </UiButton>
      </form>

      <!-- Footer -->
      <div class="auth-footer">
        <span>Don't have an account?</span>
        <router-link to="/register" class="auth-link-bold">Create one</router-link>
      </div>
    </div>
    <div class="auth-bg-decoration" />
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  position: relative;
  overflow: hidden;
}
.auth-bg-decoration {
  position: absolute;
  top: -30%; right: -20%;
  width: 60vw; height: 60vw;
  border-radius: 50%;
  background: radial-gradient(circle, oklch(0.65 0.24 265 / 0.06), transparent 70%);
  pointer-events: none;
}
.auth-container { width: 100%; max-width: 420px; padding: 2rem; position: relative; z-index: 1; }
.auth-brand { text-align: center; margin-bottom: 2rem; }
.brand-icon {
  display: inline-flex; align-items: center; justify-content: center;
  width: 56px; height: 56px; border-radius: var(--radius-lg);
  background: var(--color-primary); color: var(--color-text-on-primary);
  margin-bottom: 1rem; box-shadow: var(--shadow-glow);
}
.brand-title { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; margin-bottom: 0.25rem; }
.brand-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; }
.auth-form {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-xl);
  padding: 2rem;
  box-shadow: var(--shadow-md);
}
.auth-error {
  padding: 0.75rem 1rem;
  background: oklch(0.95 0.05 25);
  color: var(--color-danger);
  border-radius: var(--radius-md);
  font-size: 0.8125rem;
  margin-bottom: 1.25rem;
}
@media (prefers-color-scheme: dark) { .auth-error { background: oklch(0.22 0.05 25); } }
.auth-actions { display: flex; justify-content: flex-end; margin-bottom: 1.5rem; }
.auth-link { font-size: 0.8125rem; color: var(--color-primary); font-weight: 500; }
.auth-footer {
  text-align: center; margin-top: 1.5rem; font-size: 0.875rem;
  color: var(--color-text-secondary); display: flex; gap: 0.5rem; justify-content: center;
}
.auth-link-bold { color: var(--color-primary); font-weight: 600; }
</style>
