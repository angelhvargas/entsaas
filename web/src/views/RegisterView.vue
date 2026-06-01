<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { UserPlus, ArrowRight, Sparkles } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const orgName = ref('')
const error = ref('')
const loading = ref(false)

async function handleRegister() {
    error.value = ''
    loading.value = true
    try {
        await auth.register(email.value, password.value, orgName.value)
        router.push('/')
    } catch (err) {
        error.value = err.response?.data?.error?.message || 'Registration failed.'
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="auth-page">
        <div class="auth-container animate-fadeIn">
            <div class="auth-brand">
                <div class="brand-icon"><Sparkles :size="28" /></div>
                <h1 class="brand-title">Create Account</h1>
                <p class="brand-subtitle">Start building with EntSaaS</p>
            </div>

            <form @submit.prevent="handleRegister" class="auth-form">
                <div v-if="error" class="auth-error">{{ error }}</div>

                <div class="field">
                    <label class="label" for="reg-org">Organization Name</label>
                    <input id="reg-org" v-model="orgName" type="text" class="input" placeholder="Acme Inc." required />
                </div>
                <div class="field">
                    <label class="label" for="reg-email">Email</label>
                    <input id="reg-email" v-model="email" type="email" class="input" placeholder="you@company.com" required autocomplete="email" />
                </div>
                <div class="field">
                    <label class="label" for="reg-password">Password</label>
                    <input id="reg-password" v-model="password" type="password" class="input" placeholder="Min 8 characters" required minlength="8" autocomplete="new-password" />
                </div>

                <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
                    <UserPlus :size="16" />
                    {{ loading ? 'Creating...' : 'Create account' }}
                    <ArrowRight v-if="!loading" :size="16" />
                </button>
            </form>

            <div class="auth-footer">
                <span>Already have an account?</span>
                <router-link to="/login" class="auth-link-bold">Sign in</router-link>
            </div>
        </div>
        <div class="auth-bg-decoration"></div>
    </div>
</template>

<style scoped>
.auth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: var(--color-bg); position: relative; overflow: hidden; }
.auth-bg-decoration { position: absolute; top: -30%; right: -20%; width: 60vw; height: 60vw; border-radius: 50%; background: radial-gradient(circle, oklch(0.72 0.19 160 / 0.06), transparent 70%); pointer-events: none; }
.auth-container { width: 100%; max-width: 420px; padding: 2rem; position: relative; z-index: 1; }
.auth-brand { text-align: center; margin-bottom: 2rem; }
.brand-icon { display: inline-flex; align-items: center; justify-content: center; width: 56px; height: 56px; border-radius: var(--radius-lg); background: var(--color-primary); color: var(--color-text-on-primary); margin-bottom: 1rem; box-shadow: var(--shadow-glow); }
.brand-title { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; margin-bottom: 0.25rem; }
.brand-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; }
.auth-form { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-xl); padding: 2rem; box-shadow: var(--shadow-md); }
.field { margin-bottom: 1.25rem; }
.auth-error { padding: 0.75rem 1rem; background: oklch(0.95 0.05 25); color: var(--color-danger); border-radius: var(--radius-md); font-size: 0.8125rem; margin-bottom: 1.25rem; }
@media (prefers-color-scheme: dark) { .auth-error { background: oklch(0.22 0.05 25); } }
.auth-submit { width: 100%; padding: 12px; font-size: 0.9375rem; }
.auth-footer { text-align: center; margin-top: 1.5rem; font-size: 0.875rem; color: var(--color-text-secondary); display: flex; gap: 0.5rem; justify-content: center; }
.auth-link-bold { color: var(--color-primary); font-weight: 600; }
</style>
