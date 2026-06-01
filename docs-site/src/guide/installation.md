# Production & Docker Deployment

This guide outlines how to build, configure, and orchestrate the EntSaaS framework using Docker and production environment variables.

---

## 1. Single-Image Container Compilation

EntSaaS features an optimized multi-stage `Dockerfile` in the root directory. This produces a minimal, lightweight scratch container containing only the compiled Go API binary and root CA certificates.

Build the application image:
```bash
docker build -t entsaas-api:latest .
```

### Build Details:
- **Stage 1 (Builder)**: Leverages `golang:1.23-alpine`, downloads go packages, caches dependencies, and compiles with static link flags (`CGO_ENABLED=0`).
- **Stage 2 (Final Scratch)**: Copies the binary from the builder and imports standard `ca-certificates.crt` so secure outbound HTTPS requests (e.g. Stripe, Paddle, OpenAI) are supported.

---

## 2. Docker Compose Orchestration

For single-instance or local staging deployments, the root [docker-compose.yml](docker-compose.yml) configures:
- **`postgres`**: A PostgreSQL 16 server mounting persistent volumes.
- **`redis`**: A lightweight caching and streams event server.
- **`api`**: The compiled Go server hot-mounting code and listening on port `8080`.

Launch the entire stack:
```bash
docker compose up --build -d
```

---

## 3. Production Environment Checklist

When deploying to production environments (AWS, Kubernetes, GCP), ensure these environment variables are correctly injected:

### Database & Cache
- `DATABASE_URL`: Production PostgreSQL connection string (e.g., `postgres://user:pass@host:5432/db?sslmode=require`).
- `REDIS_URL`: Production Redis URL (e.g., `redis://:pass@host:6379/0`).

### Authentication
- `ENTSAAS_ENV`: Set to `production` (activates HSTS, secure cookies, and strict CSP).
- `JWT_SECRET`: A secure, high-entropy 32-byte secret key used for signing access tokens.

### Billing Gateways
- `BILLING_PROVIDER`: Either `stripe` or `paddle`.
- `STRIPE_API_KEY`: Production Stripe private key.
- `STRIPE_WEBHOOK_SECRET`: Stripe signing key for incoming event signature validation.

### Email Notifications
- `SMTP_HOST`: Production mail server (e.g., SendGrid, Mailgun).
- `SMTP_PORT`: Port (typically `587` or `465`).
- `SMTP_USER` & `SMTP_PASSWORD`: Secure credentials.
