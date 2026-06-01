# Architecture Overview

## System Diagram

```
┌────────────────────────────────────────────────────────────────────────────┐
│                          Client (Browser)                                  │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │  Vue 3 SPA                                                           │  │
│  │  ├── Router (auth guards, NProgress)                                 │  │
│  │  ├── Pinia Stores (auth, config, prefs, ai)                          │  │
│  │  ├── Axios API Client (auto refresh token rotation)                  │  │
│  │  ├── Views (Dashboard, Projects, Settings, Auth flows)               │  │
│  │  ├── Layouts (AppShell with sidebar)                                 │  │
│  │  └── AI Chat Widget (streaming SSE)                                  │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────┬───────────────────────────────────────────────┘
                             │ HTTP /v1/*
┌────────────────────────────▼───────────────────────────────────────────────┐
│                          Go API Server (Gin)                               │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │  Middleware Stack                                                     │  │
│  │  ├── Recovery (panic handler)                                        │  │
│  │  ├── Logger (request logging)                                        │  │
│  │  ├── SecurityHeaders (CSP, HSTS, X-Frame-Options)                    │  │
│  │  ├── CORS (configurable origins)                                     │  │
│  │  └── SessionAuth (JWT validation, claims injection)                  │  │
│  ├──────────────────────────────────────────────────────────────────────┤  │
│  │  Route Surfaces                                                       │  │
│  │  ├── PUBLIC  /healthz, /readyz, /v1/auth/*, /v1/config               │  │
│  │  ├── APP     /v1/me, /v1/projects/*, /v1/preferences, /v1/ai/*       │  │
│  │  └── ADMIN   /v1/admin/users/*, /v1/admin/invites/*                  │  │
│  ├──────────────────────────────────────────────────────────────────────┤  │
│  │  Handlers → Store Interface → PostgresStore                           │  │
│  │                              → RedisStore (optional)                  │  │
│  │                              → AI Provider (optional)                 │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│  SPA Fallback: NoRoute → serves web/dist/index.html                       │
└────────────────────────────┬──────────────┬────────────────────────────────┘
                             │              │
┌────────────────────────────▼──┐  ┌────────▼───────────────────────────────┐
│  PostgreSQL 18                 │  │  Redis 7 (optional)                    │
│  ├── organizations             │  │  ├── Session cache                     │
│  ├── users                     │  │  ├── Rate limiting                     │
│  ├── user_credentials          │  │  └── Eventing broker (future)          │
│  ├── refresh_tokens            │  └────────────────────────────────────────┘
│  ├── reset_tokens              │
│  ├── verification_tokens       │  ┌────────────────────────────────────────┐
│  ├── invites                   │  │  LLM Provider (optional)               │
│  ├── projects                  │  │  ├── OpenAI                            │
│  ├── audit_log                 │  │  ├── Azure OpenAI                      │
│  └── user_preferences          │  │  ├── Ollama (local)                    │
└────────────────────────────────┘  │  └── vLLM / LiteLLM                    │
                                    └────────────────────────────────────────┘
```

## Key Design Decisions

### Interface-Driven Persistence
All data access goes through the `AppStore` interface defined in `internal/store/interfaces.go`. This enables:
- Easy mocking in tests
- Swapping implementations (e.g., SQLite for testing)
- Clear contract between handlers and storage

### Route Surface Composition
Routes are organized into surfaces by auth level:
- **Public** — No auth required (health, login, register)
- **App** — Session token required (product features)
- **Admin** — Session + elevated role required (user management)

Each surface has its own registration file, keeping route tables focused and reviewable.

### JWT + Refresh Token Rotation
- Access tokens are short-lived (15min default)
- Refresh tokens are long-lived (30 days) and rotate on each use
- Old refresh tokens are revoked immediately on rotation
- All refresh tokens are revoked on password reset

### RBAC Hierarchy
```
owner (40) > admin (30) > member (20) > viewer (10)
```
Users can only manage roles strictly below their own rank.

### Audit Logging
Every state-mutating operation logs to the `audit_log` table with:
- Actor ID (who)
- Action (what)
- Entity type + ID (on what)
- Metadata (additional context)

### AI Integration
The AI layer is fully optional (controlled by `AI_ENABLED`):
- Provider interface supports any OpenAI-compatible API
- Streaming via Server-Sent Events (SSE)
- All AI interactions are audit-logged
- Frontend chat widget only renders when AI is enabled

## Data Flow

### Authentication Flow
```
Login → Verify Password → Generate JWT + Refresh Token → Return to Client
  ↓
Client stores tokens in localStorage
  ↓
Subsequent requests → Axios interceptor adds Bearer token
  ↓
401 → Interceptor uses refresh token → New JWT + rotated refresh token
```

### Request Lifecycle
```
HTTP Request
  → Gin Recovery middleware
  → Logger middleware
  → Security Headers
  → CORS
  → SessionAuth (if authenticated route)
  → RequireRole (if admin route)
  → Handler
  → Store method
  → PostgreSQL
  → JSON Response
```
