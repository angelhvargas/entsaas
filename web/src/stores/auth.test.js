import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'
import { server } from '../test/server'
import { http, HttpResponse } from 'msw'

describe('auth.js Pinia Store', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    setActivePinia(createPinia())
  })

  it('has correct default state', () => {
    const store = useAuthStore()
    expect(store.user).toBeNull()
    expect(store.organization).toBeNull()
    expect(store.accessToken).toBeNull()
    expect(store.isAuthenticated).toBe(false)
  })

  it('login action authenticates and sets state', async () => {
    server.use(
      http.post('/v1/auth/login', () => {
        return HttpResponse.json({
          access_token: 'login-access-token',
          refresh_token: 'login-refresh-token',
          user: { id: 'user-login', email: 'user@entsaas.dev', role: 'member' },
        })
      })
    )

    const store = useAuthStore()
    const data = await store.login('user@entsaas.dev', 'password')

    expect(data.access_token).toBe('login-access-token')
    expect(store.accessToken).toBe('login-access-token')
    expect(store.user).toEqual({ id: 'user-login', email: 'user@entsaas.dev', role: 'member' })
    expect(store.isAuthenticated).toBe(true)
    expect(localStorage.getItem('access_token')).toBe('login-access-token')
    expect(localStorage.getItem('refresh_token')).toBe('login-refresh-token')
  })

  it('register action creates account and sets organization state', async () => {
    server.use(
      http.post('/v1/auth/register', () => {
        return HttpResponse.json({
          access_token: 'reg-access',
          refresh_token: 'reg-refresh',
          user: { id: 'user-reg', email: 'reg@entsaas.dev' },
          organization: { id: 'org-reg', name: 'Reg Org' },
        }, { status: 201 })
      })
    )

    const store = useAuthStore()
    await store.register('reg@entsaas.dev', 'password', 'Reg Org')

    expect(store.accessToken).toBe('reg-access')
    expect(store.user.id).toBe('user-reg')
    expect(store.organization).toEqual({ id: 'org-reg', name: 'Reg Org' })
    expect(store.isAuthenticated).toBe(true)
    expect(localStorage.getItem('access_token')).toBe('reg-access')
  })

  it('fetchProfile action fetches current user profile and organization info', async () => {
    server.use(
      http.get('/v1/me', () => {
        return HttpResponse.json({
          user: { id: 'user-me', email: 'me@entsaas.dev' },
          organization: { id: 'org-me', name: 'My Org' },
        })
      })
    )

    const store = useAuthStore()
    await store.fetchProfile()

    expect(store.user.email).toBe('me@entsaas.dev')
    expect(store.organization.name).toBe('My Org')
  })

  it('logout action invokes API, clears state, and removes tokens', async () => {
    let logoutCalled = false
    server.use(
      http.post('/v1/auth/logout', () => {
        logoutCalled = true
        return HttpResponse.json({ message: 'OK' })
      })
    )

    const store = useAuthStore()
    store.accessToken = 'token-to-clear'
    store.user = { id: 'user-1' }
    store.organization = { id: 'org-1' }
    localStorage.setItem('access_token', 'token-to-clear')
    localStorage.setItem('refresh_token', 'refresh-to-clear')

    await store.logout()

    expect(logoutCalled).toBe(true)
    expect(store.accessToken).toBeNull()
    expect(store.user).toBeNull()
    expect(store.organization).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem('access_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('loginWithToken manually injects credentials', () => {
    const store = useAuthStore()
    store.loginWithToken('manual-token', { id: 'manual-user' })

    expect(store.accessToken).toBe('manual-token')
    expect(store.user).toEqual({ id: 'manual-user' })
    expect(store.isAuthenticated).toBe(true)
    expect(localStorage.getItem('access_token')).toBe('manual-token')
  })

  it('startup rehydration fetches profile if token exists', async () => {
    localStorage.setItem('access_token', 'existing-token')

    let meCalled = false
    server.use(
      http.get('/v1/me', () => {
        meCalled = true
        return HttpResponse.json({
          user: { id: 'user-hydrate' },
          organization: { id: 'org-hydrate' },
        })
      })
    )

    const store = useAuthStore()
    
    // Give happy-dom microtasks a tick to complete the rehydration promise chain
    await new Promise((r) => setTimeout(r, 0))

    expect(meCalled).toBe(true)
    expect(store.user).toEqual({ id: 'user-hydrate' })
    expect(store.organization).toEqual({ id: 'org-hydrate' })
  })
})
