<script setup>
import { ref } from 'vue'
import { Mail, ArrowLeft, Sparkles } from 'lucide-vue-next'
import client from '@/api/client'

const email = ref('')
const submitted = ref(false)
const loading = ref(false)

async function handleSubmit() {
    loading.value = true
    try { await client.post('/auth/forgot-password', { email: email.value }) } catch {}
    submitted.value = true
    loading.value = false
}
</script>

<template>
    <div class="auth-page">
        <div class="auth-container animate-fadeIn">
            <div class="auth-brand">
                <div class="brand-icon"><Sparkles :size="28" /></div>
                <h1 class="brand-title">Reset Password</h1>
                <p class="brand-subtitle">We'll send you a reset link</p>
            </div>

            <div v-if="submitted" class="auth-form" style="text-align:center;padding:2.5rem 2rem">
                <Mail :size="40" style="color:var(--color-primary);margin-bottom:1rem" />
                <p style="font-weight:500;margin-bottom:0.5rem">Check your email</p>
                <p style="color:var(--color-text-secondary);font-size:0.875rem">If an account exists, a reset link has been sent.</p>
            </div>
            <form v-else @submit.prevent="handleSubmit" class="auth-form">
                <div class="field">
                    <label class="label" for="fp-email">Email</label>
                    <input id="fp-email" v-model="email" type="email" class="input" placeholder="you@company.com" required />
                </div>
                <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
                    {{ loading ? 'Sending...' : 'Send reset link' }}
                </button>
            </form>

            <div class="auth-footer">
                <router-link to="/login" class="auth-link-bold"><ArrowLeft :size="14" style="vertical-align:middle" /> Back to login</router-link>
            </div>
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
.brand-title { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; margin-bottom: 0.25rem; }
.brand-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; }
.auth-form { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-xl); padding: 2rem; box-shadow: var(--shadow-md); }
.field { margin-bottom: 1.25rem; }
.auth-submit { width: 100%; padding: 12px; font-size: 0.9375rem; }
.auth-footer { text-align: center; margin-top: 1.5rem; }
.auth-link-bold { color: var(--color-primary); font-weight: 600; font-size: 0.875rem; }
</style>
