<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { User, Shield, Bell, CreditCard } from 'lucide-vue-next'
import { UiCard, UiTabs, UiBadge, UiSwitch, UiButton, UiField, UiInput } from '@/components/ui'
import { useRouter } from 'vue-router'

const auth   = useAuthStore()
const router = useRouter()
const activeTab = ref('profile')
const tabs = [
  { key: 'profile',       label: 'Profile',       icon: User },
  { key: 'security',      label: 'Security',       icon: Shield },
  { key: 'notifications', label: 'Notifications',  icon: Bell },
  { key: 'billing',       label: 'Billing',        icon: CreditCard },
]

// Notification prefs state (UI only for now)
const emailOnInvite = ref(true)
const emailOnProject = ref(false)
</script>

<template>
  <div class="animate-fadeIn">
    <div class="page-header">
      <div>
        <h1 class="page-title">Settings</h1>
        <p class="page-subtitle">Manage your account and organization</p>
      </div>
    </div>

    <UiTabs v-model="activeTab" :tabs="tabs">
      <!-- Profile tab -->
      <div v-if="activeTab === 'profile'">
        <UiCard>
          <template #header>
            <div class="section-header-row">
              <User :size="18" style="color:var(--color-primary)" /> Profile Information
            </div>
          </template>
          <div class="field-list">
            <div class="field-row">
              <span class="field-label">Email</span>
              <span class="field-value">{{ auth.user?.email }}</span>
            </div>
            <div class="field-row">
              <span class="field-label">Role</span>
              <UiBadge :variant="auth.user?.role === 'owner' ? 'primary' : 'default'" size="sm">
                {{ auth.user?.role }}
              </UiBadge>
            </div>
            <div class="field-row">
              <span class="field-label">Organization</span>
              <span class="field-value">{{ auth.organization?.name || '—' }}</span>
            </div>
            <div class="field-row">
              <span class="field-label">Member since</span>
              <span class="field-value">{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString() : '—' }}</span>
            </div>
          </div>
        </UiCard>
      </div>

      <!-- Security tab -->
      <div v-else-if="activeTab === 'security'">
        <UiCard>
          <template #header>
            <div class="section-header-row">
              <Shield :size="18" style="color:var(--color-primary)" /> Change Password
            </div>
          </template>
          <form class="security-form" @submit.prevent>
            <UiField label="Current password" html-for="curr-pw">
              <UiInput id="curr-pw" type="password" placeholder="••••••••" />
            </UiField>
            <UiField label="New password" html-for="new-pw">
              <UiInput id="new-pw" type="password" placeholder="••••••••" />
            </UiField>
            <UiField label="Confirm new password" html-for="conf-pw">
              <UiInput id="conf-pw" type="password" placeholder="••••••••" />
            </UiField>
            <UiButton type="submit" variant="primary">Update password</UiButton>
          </form>
        </UiCard>
      </div>

      <!-- Notifications tab -->
      <div v-else-if="activeTab === 'notifications'">
        <UiCard>
          <template #header>
            <div class="section-header-row">
              <Bell :size="18" style="color:var(--color-primary)" /> Email Notifications
            </div>
          </template>
          <div class="field-list">
            <div class="field-row">
              <div>
                <div class="field-value">Team invitations</div>
                <div class="field-label">Receive an email when you're invited to a project</div>
              </div>
              <UiSwitch v-model="emailOnInvite" />
            </div>
            <div class="field-row">
              <div>
                <div class="field-value">Project updates</div>
                <div class="field-label">Get notified when projects change status</div>
              </div>
              <UiSwitch v-model="emailOnProject" />
            </div>
          </div>
          <template #footer>
            <UiButton variant="primary" size="sm">Save preferences</UiButton>
          </template>
        </UiCard>
      </div>
      <!-- Billing tab -->
      <div v-else-if="activeTab === 'billing'">
        <UiCard padding="md">
          <template #header>
            <div class="section-header-row">
              <CreditCard :size="18" style="color:var(--color-primary)" /> Billing &amp; Subscription
            </div>
          </template>
          <p style="font-size:0.875rem;color:var(--color-text-secondary);margin-bottom:1.5rem;">
            Manage your plan, view invoices, and update payment details.
          </p>
          <template #footer>
            <UiButton variant="primary" size="sm" @click="router.push('/settings/billing')">
              <CreditCard :size="14" /> Go to Billing
            </UiButton>
          </template>
        </UiCard>
      </div>
    </UiTabs>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 2rem; }
.page-title { font-size: 1.5rem; font-weight: 700; letter-spacing: -0.02em; }
.page-subtitle { color: var(--color-text-secondary); font-size: 0.9375rem; margin-top: 0.25rem; }

.section-header-row { display: flex; align-items: center; gap: 8px; }
.field-list { }
.field-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--color-border-subtle);
  gap: 12px;
}
.field-row:last-child { border-bottom: none; }
.field-label { font-size: 0.8125rem; color: var(--color-text-secondary); margin-top: 2px; }
.field-value { font-size: 0.875rem; font-weight: 500; }

.security-form { display: flex; flex-direction: column; gap: 0; max-width: 400px; }
</style>
