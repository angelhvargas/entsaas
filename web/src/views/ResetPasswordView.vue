<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { KeyRound, Sparkles } from 'lucide-vue-next'
import client from '@/api/client'

const route = useRoute()
const router = useRouter()
const password = ref('')
const confirm = ref('')
const error = ref('')
const success = ref(false)
const loading = ref(false)

async function handleReset() {
    if (password.value !== confirm.value) { error.value = 'Passwords do not match'; return }
    error.value = ''
    loading.value = true
    try {
        await client.post('/auth/reset-password', { token: route.query.token, password: password.value })
        success.value = true
        setTimeout(() => router.push('/login'), 2000)
    } catch (err) {
        error.value = err.response?.data?.error?.message || 'Reset failed.'
    } finally { loading.value = false }
}
</script>

<template>
    <div class="auth-page">
        <div class="auth-container animate-fadeIn">
            <div class="auth-brand">
                <div class="brand-icon"><Sparkles :size="28" /></div>
                <h1 class="brand-title">New Password</h1>
            </div>
            <div v-if="success" class="auth-form" style="text-align:center;padding:2.5rem 2rem">
                <KeyRound :size="40" style="color:var(--color-success);margin-bottom:1rem" />
                <p style="font-weight:500">Password reset successfully</p>
                <p style="color:var(--color-text-secondary);font-size:0.875rem;margin-top:0.5rem">Redirecting to login...</p>
            </div>
            <form v-else @submit.prevent="handleReset" class="auth-form">
                <div v-if="error" class="auth-error">{{ error }}</div>
                <div class="field">
                    <label class="label" for="rp-pass">New Password</label>
                    <input id="rp-pass" v-model="password" type="password" class="input" placeholder="Min 8 characters" required minlength="8" />
                </div>
                <div class="field">
                    <label class="label" for="rp-confirm">Confirm Password</label>
                    <input id="rp-confirm" v-model="confirm" type="password" class="input" placeholder="Repeat password" required />
                </div>
                <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">{{ loading ? 'Resetting...' : 'Reset password' }}</button>
            </form>
        </div>
        <div class="auth-bg-decoration"></div>
    </div>
</template>

<style scoped>
.auth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: var(--color-bg); position: relative; overflow: hidden; }
.auth-bg-decoration { position: absolute; top: -30%; right: -20%; width: 60vw; height: 60vw; border-radius: 50%; background: radial-gradient(circle, oklch(0.65 0.24 265 / 0.06), transparent 70%); pointer-events: none; }
.auth-container { width: 100%; max-width: 420px; padding: 2rem; position: relative; z-index: 1; }
.auth-brand { text-align: center; margin-bottom: 2rem; }
.brand-icon { display: inline-flex; align-items: center; justify-content: center; width: 56px; height: 56px; border-radius: var(--radius-lg); background: var(--color-primary); color: var(--color-text-on-primary); margin-bottom: 1rem; box-shadow: var(--shadow-glow); }
.brand-title { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; }
.auth-form { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-xl); padding: 2rem; box-shadow: var(--shadow-md); }
.field { margin-bottom: 1.25rem; }
.auth-error { padding: 0.75rem 1rem; background: oklch(0.95 0.05 25); color: var(--color-danger); border-radius: var(--radius-md); font-size: 0.8125rem; margin-bottom: 1.25rem; }
@media (prefers-color-scheme: dark) { .auth-error { background: oklch(0.22 0.05 25); } }
.auth-submit { width: 100%; padding: 12px; font-size: 0.9375rem; }
</style>
