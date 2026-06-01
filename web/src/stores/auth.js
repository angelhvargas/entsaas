import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import client from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
    const user = ref(null)
    const organization = ref(null)
    const accessToken = ref(localStorage.getItem('access_token'))

    const isAuthenticated = computed(() => !!accessToken.value)

    async function login(email, password) {
        const { data } = await client.post('/auth/login', { email, password })
        accessToken.value = data.access_token
        localStorage.setItem('access_token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)
        user.value = data.user
        return data
    }

    async function register(email, password, orgName) {
        const { data } = await client.post('/auth/register', {
            email,
            password,
            org_name: orgName,
        })
        accessToken.value = data.access_token
        localStorage.setItem('access_token', data.access_token)
        if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
        user.value = data.user
        organization.value = data.organization
        return data
    }

    async function fetchProfile() {
        const { data } = await client.get('/me')
        user.value = data.user
        organization.value = data.organization
        return data
    }

    async function logout() {
        try { await client.post('/auth/logout') } catch {}
        accessToken.value = null
        user.value = null
        organization.value = null
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
    }

    // Rehydrate on startup
    if (accessToken.value) {
        fetchProfile().catch(() => logout())
    }

    // Used by AcceptInviteView and similar flows that receive a token directly.
    function loginWithToken(token, userData) {
        accessToken.value = token
        localStorage.setItem('access_token', token)
        user.value = userData
    }

    return { user, organization, accessToken, isAuthenticated, login, register, fetchProfile, logout, loginWithToken }
})
