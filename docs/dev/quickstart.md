# Quick Start

Get EntSaaS running locally in 5 minutes.

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker and Docker Compose
- Make

## Steps

### 1. Clone and configure

```bash
git clone <repo-url> && cd entsaas
cp .env.dev .env
```

### 2. Start infrastructure

```bash
# Start Postgres + Redis + API + Vite dev server
make dev-up
```

### 3. Run migrations

```bash
make migrate
```

### 4. Seed admin user

```bash
make seed-dev
```

### 5. Open the dashboard

```
http://localhost:5173
```

Login credentials:
- **Email:** `admin@entsaas.dev`
- **Password:** `password`

## What's Running

| Service | URL | Description |
|---------|-----|-------------|
| Vue Dashboard | http://localhost:5173 | Vite dev server with HMR |
| Go API | http://localhost:8080 | Gin HTTP server |
| PostgreSQL | localhost:5432 | Primary database |
| Redis | localhost:6379 | Cache (dev profile) |

## Next Steps

- Read [Extending EntSaaS](extending.md) to add your first feature
- Review [Environment Variables](env-vars.md) for configuration options
- Check [Architecture Overview](../architecture/overview.md) for system design
