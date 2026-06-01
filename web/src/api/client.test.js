import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import client from './client'
import { server } from '../test/server'
import { http, HttpResponse } from 'msw'

describe('client.js (API Client Interceptors)', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    
    // Stub window.location safely with a proper base URL and reactive href
    let mockHref = 'http://localhost/'
    const locationMock = {
      get href() { return mockHref },
      set href(val) { mockHref = val },
      origin: 'http://localhost',
      protocol: 'http:',
      host: 'localhost',
      hostname: 'localhost',
      port: '',
      pathname: '/',
      search: '',
      hash: '',
      assign: vi.fn(),
      replace: vi.fn()
    }
    vi.stubGlobal('location', locationMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('injects access_token from localStorage into Request headers', async () => {
    localStorage.setItem('access_token', 'test-token-123')

    // Define a dummy endpoint to inspect headers
    let capturedHeaders = {}
    server.use(
      http.get('/v1/dummy', ({ request }) => {
        capturedHeaders = Object.fromEntries(request.headers.entries())
        return HttpResponse.json({ status: 'ok' })
      })
    )

    await client.get('/dummy')

    expect(capturedHeaders.Authorization).toBe('Bearer test-token-123')
  })

  it('does not inject Authorization header if no access_token is present', async () => {
    let capturedHeaders = {}
    server.use(
      http.get('/v1/dummy', ({ request }) => {
        capturedHeaders = Object.fromEntries(request.headers.entries())
        return HttpResponse.json({ status: 'ok' })
      })
    )

    await client.get('/dummy')

    expect(capturedHeaders.authorization).toBeUndefined()
  })

  it('performs automatic token refresh on 401 Unauthorized and retries the request', async () => {
    localStorage.setItem('access_token', 'expired-token')
    localStorage.setItem('refresh_token', 'my-refresh-token')

    let requestCount = 0
    let refreshTriggered = false

    // Intercept refresh and dummy endpoint
    server.use(
      http.post('/v1/auth/refresh', async ({ request }) => {
        const body = await request.json()
        expect(body.refresh_token).toBe('my-refresh-token')
        refreshTriggered = true
        return HttpResponse.json({
          access_token: 'fresh-access-token',
          refresh_token: 'fresh-refresh-token',
        })
      }),
      http.get('/v1/dummy', ({ request }) => {
        requestCount++
        const auth = request.headers.get('authorization')
        if (requestCount === 1) {
          // First time, simulate 401
          expect(auth).toBe('Bearer expired-token')
          return HttpResponse.json({ error: 'Unauthorized' }, { status: 401 })
        }
        // Second time (retry), expect fresh token and return 200
        expect(auth).toBe('Bearer fresh-access-token')
        return HttpResponse.json({ data: 'success' })
      })
    )

    const res = await client.get('/dummy')

    expect(refreshTriggered).toBe(true)
    expect(res.data.data).toBe('success')
    expect(localStorage.getItem('access_token')).toBe('fresh-access-token')
    expect(localStorage.getItem('refresh_token')).toBe('fresh-refresh-token')
  })

  it('buffers concurrent 401 requests and retries all of them once refresh succeeds', async () => {
    localStorage.setItem('access_token', 'expired-token')
    localStorage.setItem('refresh_token', 'my-refresh-token')

    let refreshCalls = 0
    let get1Count = 0
    let get2Count = 0

    server.use(
      http.post('/v1/auth/refresh', async () => {
        refreshCalls++
        // Simulate network delay for refresh
        await new Promise((r) => setTimeout(r, 10))
        return HttpResponse.json({
          access_token: 'new-access',
          refresh_token: 'new-refresh',
        })
      }),
      http.get('/v1/dummy-1', ({ request }) => {
        get1Count++
        if (get1Count === 1) return HttpResponse.json({}, { status: 401 })
        expect(request.headers.get('authorization')).toBe('Bearer new-access')
        return HttpResponse.json({ val: 1 })
      }),
      http.get('/v1/dummy-2', ({ request }) => {
        get2Count++
        if (get2Count === 1) return HttpResponse.json({}, { status: 401 })
        expect(request.headers.get('authorization')).toBe('Bearer new-access')
        return HttpResponse.json({ val: 2 })
      })
    )

    // Trigger two requests concurrently
    const [res1, res2] = await Promise.all([
      client.get('/dummy-1'),
      client.get('/dummy-2'),
    ])

    expect(refreshCalls).toBe(1) // Only ONE refresh call should occur!
    expect(res1.data.val).toBe(1)
    expect(res2.data.val).toBe(2)
  })

  it('fails refresh, clears tokens, and redirects to login on refresh error', async () => {
    localStorage.setItem('access_token', 'expired')
    localStorage.setItem('refresh_token', 'refresh')

    server.use(
      http.post('/v1/auth/refresh', () => {
        return HttpResponse.json({ error: 'invalid refresh token' }, { status: 400 })
      }),
      http.get('/v1/dummy', () => {
        return HttpResponse.json({}, { status: 401 })
      })
    )

    await expect(client.get('/dummy')).rejects.toThrow()

    expect(localStorage.getItem('access_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(window.location.href).toBe('/login')
  })
})
