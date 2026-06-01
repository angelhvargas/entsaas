/**
 * src/test/handlers.js
 *
 * MSW API mock handlers for all entsaas /api/v1/* endpoints.
 * Import and use `server.use(handler)` inside individual tests to override defaults.
 */
import { http, HttpResponse } from 'msw'

// Must match the axios client baseURL in src/api/client.js
const BASE = '/v1'

// ── Auth ─────────────────────────────────────────────────────────────────────

export const authHandlers = [
  http.post(`${BASE}/auth/login`, () =>
    HttpResponse.json({
      access_token: 'test-access-token',
      refresh_token: 'test-refresh-token',
      token_type: 'Bearer',
      expires_in: 900,
      user: { id: 'user-1', email: 'admin@entsaas.dev', role: 'owner', org_id: 'org-1' },
      organization: { id: 'org-1', name: 'Test Org', slug: 'test-org' },
    })
  ),

  http.post(`${BASE}/auth/register`, () =>
    HttpResponse.json(
      {
        access_token: 'test-access-token',
        token_type: 'Bearer',
        expires_in: 900,
        user: { id: 'user-2', email: 'new@entsaas.dev', role: 'owner', org_id: 'org-2' },
        organization: { id: 'org-2', name: 'New Org', slug: 'new-org' },
      },
      { status: 201 }
    )
  ),

  http.post(`${BASE}/auth/logout`, () => HttpResponse.json({ message: 'Logged out' })),

  http.get(`${BASE}/me`, () =>
    HttpResponse.json({
      user: { id: 'user-1', email: 'admin@entsaas.dev', role: 'owner', org_id: 'org-1' },
      organization: { id: 'org-1', name: 'Test Org', slug: 'test-org' },
    })
  ),
]

// ── Projects ──────────────────────────────────────────────────────────────────

export const projectHandlers = [
  http.get(`${BASE}/projects`, () =>
    HttpResponse.json({
      projects: [
        { id: 'proj-1', name: 'Alpha', org_id: 'org-1', status: 'active', created_at: new Date().toISOString() },
        { id: 'proj-2', name: 'Beta',  org_id: 'org-1', status: 'active', created_at: new Date().toISOString() },
      ],
    })
  ),

  http.post(`${BASE}/projects`, async ({ request }) => {
    const body = await request.json()
    return HttpResponse.json(
      { project: { id: 'proj-new', name: body.name, org_id: 'org-1', status: 'active', created_at: new Date().toISOString() } },
      { status: 201 }
    )
  }),

  http.delete(`${BASE}/projects/:id`, () => HttpResponse.json({ message: 'Project deleted' })),
]

// ── Billing ───────────────────────────────────────────────────────────────────

export const billingHandlers = [
  http.get(`${BASE}/billing/plans`, () =>
    HttpResponse.json({
      plans: [
        { id: 'free', name: 'Free', description: 'Up to 3 projects', price_monthly: 0, currency: 'usd', is_active: true, features: ['3 projects', '3 members'] },
        { id: 'pro',  name: 'Pro',  description: 'Unlimited projects + AI', price_monthly: 49, currency: 'usd', is_active: true, features: ['Unlimited projects', 'AI assistant', 'Priority support'] },
      ],
    })
  ),

  http.get(`${BASE}/billing/subscription`, () =>
    HttpResponse.json({
      subscription: {
        id: 'sub-1',
        org_id: 'org-1',
        plan_id: 'free',
        status: 'active',
        current_period_start: new Date().toISOString(),
        current_period_end: new Date(Date.now() + 30 * 86400_000).toISOString(),
        cancel_at_period_end: false,
      },
    })
  ),

  http.post(`${BASE}/billing/checkout`, async ({ request }) => {
    const body = await request.json()
    return HttpResponse.json({ checkout_url: `https://checkout.example.com?plan=${body.plan_id}`, session_id: 'cs_test' })
  }),
]

// ── Invites ───────────────────────────────────────────────────────────────────

export const inviteHandlers = [
  http.get(`${BASE}/invites/peek`, ({ request }) => {
    const url = new URL(request.url)
    const token = url.searchParams.get('token')
    if (token === 'valid-token') {
      return HttpResponse.json({
        invite: { email: 'invited@example.com', role: 'member', org_name: 'Test Org', expires_at: new Date(Date.now() + 86400_000).toISOString() },
      })
    }
    return HttpResponse.json({ error: { code: 'INVITE_NOT_FOUND', message: 'Invite not found or expired.' } }, { status: 404 })
  }),

  http.post(`${BASE}/invites/accept`, async ({ request }) => {
    const body = await request.json()
    if (body.token === 'valid-token') {
      return HttpResponse.json(
        {
          access_token: 'invited-user-token',
          token_type: 'Bearer',
          expires_in: 900,
          user: { id: 'user-invited', email: 'invited@example.com', role: 'member', org_id: 'org-1' },
        },
        { status: 201 }
      )
    }
    return HttpResponse.json({ error: { code: 'INVITE_INVALID', message: 'Invite is invalid or has expired.' } }, { status: 400 })
  }),
]

// ── Health ────────────────────────────────────────────────────────────────────

export const healthHandlers = [
  http.get(`${BASE}/health`, () => HttpResponse.json({ status: 'ok', version: 'test' })),
  http.get(`${BASE}/config`,  () => HttpResponse.json({ features: { ai: false, billing: true } })),
]

// ── All handlers (default) ────────────────────────────────────────────────────

export const handlers = [
  ...authHandlers,
  ...projectHandlers,
  ...billingHandlers,
  ...inviteHandlers,
  ...healthHandlers,
]
