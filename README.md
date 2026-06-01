# EntSaaS

> Generic Go + Vue.js SaaS framework for bootstrapping AI-integrated solutions.

EntSaaS extracts battle-tested patterns from production SaaS systems into a clean, configurable, documentation-first framework. Start building your product in minutes, not months.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│  Frontend — Vue 3 + Vite + Pinia + Tailwind + Lucide Icons      │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────────────┐ │
│  │ Router  │ │ Stores   │ │ API      │ │ UI Component        │ │
│  │ + Guards│ │ (Pinia)  │ │ Client   │ │ Library             │ │
│  └─────────┘ └──────────┘ └──────────┘ └─────────────────────┘ │
└───────────────────────────┬──────────────────────────────────────┘
                            │ /v1/* (proxied by Vite in dev)
┌───────────────────────────▼──────────────────────────────────────┐
│  Backend — Go + Gin                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────────┐│
│  │ Router   │ │Middleware│ │ Handlers │ │ Store Interfaces     ││
│  │ (public, │ │ (JWT,    │ │ (auth,   │ │ (Postgres,           ││
│  │  app,    │ │  CORS,   │ │  CRUD,   │ │  Redis, JWT)         ││
│  │  admin)  │ │  RBAC)   │ │  prefs)  │ │                      ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────────┘│
└───────────────────────────┬──────────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│  Infrastructure                                                   │
│  ┌─────────────┐  ┌──────────┐  ┌──────────────────────────────┐│
│  │ PostgreSQL  │  │  Redis   │  │  Goose Migrations            ││
│  └─────────────┘  └──────────┘  └──────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
```

## Quick Start

```bash
# 1. Clone and configure
git clone <repo> && cd entsaas
cp .env.example .env

# 2. Start the dev stack (Postgres + Redis + API + Vite)
make dev-up

# 3. Run migrations and seed
make migrate
make seed-dev

# 4. Open the dashboard
open http://localhost:5173
# Login: admin@entsaas.dev / password
```

## Project Structure

```
entsaas/
├── cmd/
│   ├── api/          # HTTP server entry point
│   ├── migrate/      # Database migration runner
│   └── seed/         # Admin user seeder
├── internal/
│   ├── api/          # Router composition + route surfaces
│   ├── auth/         # RBAC definitions
│   ├── bootstrap/    # Shared config/env helpers
│   ├── handlers/     # HTTP request handlers
│   ├── middleware/    # JWT auth, CORS, security headers
│   ├── migrations/   # SQL migration files (Goose)
│   └── store/        # Data access (Postgres, Redis, JWT)
├── web/
│   ├── src/
│   │   ├── api/      # Axios client with auth interceptors
│   │   ├── layouts/  # AppShell (sidebar + content)
│   │   ├── router/   # Vue Router with auth guards
│   │   ├── stores/   # Pinia stores (auth, config, prefs)
│   │   └── views/    # Page components
│   └── vite.config.js
├── deploy/           # Docker init scripts
├── docs/             # Documentation
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## Features

- **Multi-tenant authentication** — JWT + refresh token rotation, RBAC (owner/admin/member/viewer)
- **Complete auth flows** — Login, register, forgot/reset password, email verification
- **Project management** — CRUD with org-scoped isolation
- **Audit logging** — Every mutation is logged with actor, action, and metadata
- **User preferences** — JSON-based per-user settings
- **Invite system** — Email-based team invites with expiry
- **Security headers** — CSP, HSTS, X-Frame-Options, XSS protection
- **Premium UI** — Dark mode, oklch colors, Inter typography, Lucide icons, micro-animations
- **Dev experience** — Docker Compose, hot reload, API proxy, comprehensive Makefile

## Make Targets

| Target | Description |
|--------|-------------|
| `make dev` | Start dev stack (foreground) |
| `make dev-up` | Start dev stack (detached) |
| `make migrate` | Run database migrations |
| `make seed-dev` | Seed admin user + test data |
| `make test` | Run Go unit tests |
| `make test-web` | Run Vue unit tests |
| `make build` | Build everything for production |
| `make check-local` | Full local quality check |

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25+, Gin, pgx, zerolog |
| Frontend | Vue 3, Vite 8, Pinia 3, Tailwind 4 |
| Database | PostgreSQL 18, Redis 7 |
| Auth | JWT (HS256), bcrypt, refresh token rotation |
| Migrations | Goose v3 (embedded SQL) |
| Icons | Lucide Vue Next |
| Build | Multi-stage Docker, Alpine 3.21 |

## License

Proprietary — All rights reserved.
