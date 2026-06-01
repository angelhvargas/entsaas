<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBillingStore } from '@/stores/billing'
import {
    LayoutDashboard, FolderKanban, Settings, LogOut, Sparkles,
    ChevronDown, Menu, X, CreditCard, BarChart2
} from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()
const billingStore = useBillingStore()

const sidebarOpen = ref(true)
const profileOpen = ref(false)

const planBadge = computed(() => {
    const map = { pro: 'Pro', enterprise: 'Ent' }
    return map[billingStore.planSlug] ?? null
})

const navItems = [
    { name: 'Dashboard', icon: LayoutDashboard, to: '/' },
    { name: 'Projects', icon: FolderKanban, to: '/projects' },
    { name: 'Plan & Usage', icon: BarChart2, to: '/settings/plan', badge: planBadge },
    { name: 'Billing', icon: CreditCard, to: '/settings/billing' },
    { name: 'Settings', icon: Settings, to: '/settings' },
]

onMounted(() => {
    billingStore.fetchPlan()
})

async function logout() {
    await auth.logout()
    billingStore.$reset()
    router.push('/login')
}
</script>

<template>
    <div class="shell" :class="{ 'sidebar-collapsed': !sidebarOpen }">
        <!-- Sidebar -->
        <aside class="sidebar">
            <div class="sidebar-header">
                <div class="sidebar-brand">
                    <Sparkles :size="22" class="brand-sparkle" />
                    <span v-if="sidebarOpen" class="brand-text">EntSaaS</span>
                </div>
                <button class="btn-ghost sidebar-toggle" @click="sidebarOpen = !sidebarOpen">
                    <component :is="sidebarOpen ? X : Menu" :size="18" />
                </button>
            </div>

            <nav class="sidebar-nav">
                <router-link
                    v-for="item in navItems"
                    :key="item.to"
                    :to="item.to"
                    class="nav-item"
                    active-class="nav-item--active"
                    :title="item.name"
                >
                    <component :is="item.icon" :size="20" />
                    <span v-if="sidebarOpen" class="nav-label">{{ item.name }}</span>
                    <span
                        v-if="sidebarOpen && item.badge?.value"
                        class="nav-plan-badge"
                    >{{ item.badge.value }}</span>
                </router-link>
            </nav>

            <div class="sidebar-footer">
                <div class="user-pill" @click="profileOpen = !profileOpen" v-click-outside="() => profileOpen = false">
                    <div class="avatar">{{ auth.user?.email?.[0]?.toUpperCase() || '?' }}</div>
                    <template v-if="sidebarOpen">
                        <div class="user-info">
                            <div class="user-email">{{ auth.user?.email }}</div>
                            <div class="user-role">{{ auth.user?.role }}</div>
                        </div>
                        <ChevronDown :size="14" />
                    </template>

                    <div v-if="profileOpen" class="profile-dropdown">
                        <button class="dropdown-item" @click="logout">
                            <LogOut :size="16" /> Sign out
                        </button>
                    </div>
                </div>
            </div>
        </aside>

        <!-- Main content -->
        <main class="main-content">
            <router-view />
        </main>
    </div>
</template>

<style scoped>
.shell {
    display: flex;
    min-height: 100vh;
}
.sidebar {
    width: var(--sidebar-width);
    background: var(--color-bg-elevated);
    border-right: 1px solid var(--color-border-subtle);
    display: flex;
    flex-direction: column;
    transition: width var(--transition-normal);
    position: sticky;
    top: 0;
    height: 100vh;
    overflow-y: auto;
}
.sidebar-collapsed .sidebar { width: 68px; }

.sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    border-bottom: 1px solid var(--color-border-subtle);
}
.sidebar-brand {
    display: flex;
    align-items: center;
    gap: 10px;
}
.brand-sparkle { color: var(--color-primary); }
.brand-text {
    font-weight: 700;
    font-size: 1.125rem;
    letter-spacing: -0.02em;
}
.sidebar-toggle {
    padding: 6px;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    cursor: pointer;
    background: none;
    border: none;
}
.sidebar-toggle:hover { background: var(--color-bg-muted); }

.sidebar-nav {
    flex: 1;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 2px;
}
.nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: var(--radius-md);
    color: var(--color-text-secondary);
    font-size: 0.875rem;
    font-weight: 500;
    transition: all var(--transition-fast);
    text-decoration: none;
}
.nav-item:hover {
    background: var(--color-bg-muted);
    color: var(--color-text);
}
.nav-item--active {
    background: var(--color-primary-subtle);
    color: var(--color-primary);
}
.nav-label { white-space: nowrap; flex: 1; }
.nav-plan-badge {
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    padding: 2px 5px;
    border-radius: 4px;
    background: var(--color-primary);
    color: var(--color-text-on-primary);
    line-height: 1;
    text-transform: uppercase;
    margin-left: auto;
}

.sidebar-footer {
    padding: 12px 8px;
    border-top: 1px solid var(--color-border-subtle);
}
.user-pill {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border-radius: var(--radius-md);
    cursor: pointer;
    position: relative;
    transition: background var(--transition-fast);
}
.user-pill:hover { background: var(--color-bg-muted); }
.avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--color-primary);
    color: var(--color-text-on-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 0.8125rem;
    flex-shrink: 0;
}
.user-info { flex: 1; min-width: 0; }
.user-email { font-size: 0.8125rem; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.user-role { font-size: 0.6875rem; color: var(--color-text-tertiary); text-transform: capitalize; }

.profile-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    right: 0;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-md);
    padding: 4px;
    margin-bottom: 4px;
    z-index: 100;
    animation: fadeIn 0.15s ease-out;
}
.dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    border: none;
    background: none;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    font-size: 0.8125rem;
    cursor: pointer;
    transition: all var(--transition-fast);
}
.dropdown-item:hover {
    background: var(--color-bg-muted);
    color: var(--color-danger);
}

.main-content {
    flex: 1;
    padding: 32px;
    max-width: 1200px;
}

@media (max-width: 768px) {
    .sidebar { width: 68px; }
    .nav-label, .brand-text, .user-info { display: none; }
    .main-content { padding: 20px; }
}
</style>
