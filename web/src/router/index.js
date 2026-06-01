import { createRouter, createWebHistory } from 'vue-router'
import NProgress from 'nprogress'

const routes = [
    // ── Public routes ────────────────────────────────────────────────────────
    {
        path: '/login',
        name: 'login',
        component: () => import('@/views/LoginView.vue'),
        meta: { guest: true },
    },
    {
        path: '/register',
        name: 'register',
        component: () => import('@/views/RegisterView.vue'),
        meta: { guest: true },
    },
    {
        path: '/forgot-password',
        name: 'forgot-password',
        component: () => import('@/views/ForgotPasswordView.vue'),
        meta: { guest: true },
    },
    {
        path: '/reset-password',
        name: 'reset-password',
        component: () => import('@/views/ResetPasswordView.vue'),
        meta: { guest: true },
    },
    {
        path: '/accept-invite',
        name: 'accept-invite',
        component: () => import('@/views/AcceptInviteView.vue'),
        meta: { guest: true },
    },

    // ── Authenticated routes ─────────────────────────────────────────────────
    {
        path: '/',
        component: () => import('@/layouts/AppShell.vue'),
        meta: { requiresAuth: true },
        children: [
            {
                path: '',
                name: 'dashboard',
                component: () => import('@/views/DashboardView.vue'),
            },
            {
                path: 'projects',
                name: 'projects',
                component: () => import('@/views/ProjectsView.vue'),
            },
            {
                path: 'settings',
                name: 'settings',
                component: () => import('@/views/SettingsView.vue'),
            },
            {
                path: 'settings/billing',
                name: 'billing',
                component: () => import('@/views/BillingView.vue'),
            },
            {
                path: 'settings/plan',
                name: 'plan-usage',
                component: () => import('@/views/PlanUsageView.vue'),
            },
            {
                path: 'settings/billing/checkout/success',
                name: 'checkout-success',
                component: () => import('@/views/CheckoutReturnView.vue'),
            },
            {
                path: 'settings/billing/checkout/cancel',
                name: 'checkout-cancel',
                component: () => import('@/views/CheckoutReturnView.vue'),
            },
        ],
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

// ── Navigation guards ───────────────────────────────────────────────────────
router.beforeEach((to, from, next) => {
    NProgress.start()
    const token = localStorage.getItem('access_token')

    if (to.meta.requiresAuth && !token) {
        return next({ name: 'login', query: { redirect: to.fullPath } })
    }

    if (to.meta.guest && token) {
        return next({ name: 'dashboard' })
    }

    next()
})

router.afterEach(() => {
    NProgress.done()
})

export default router
