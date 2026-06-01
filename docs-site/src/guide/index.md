# Introduction to EntSaaS

EntSaaS is a production-ready, highly extensible full-stack framework boilerplate designed to accelerate the development of secure, scalable, and multi-tenant software-as-a-service (SaaS) products. 

By combining the speed and type safety of **Go (Golang)** on the backend with the reactivity of **Vue 3 & Pinia** on the frontend, EntSaaS eliminates the initial 2-3 months of boilerplate development required to ship a commercial software product.

---

## High-Level Tech Stack

### Backend Engine
- **Core Server**: Go (Golang) featuring the high-performance **Gin** HTTP router.
- **Database**: PostgreSQL segmenting data through dynamic connection pooling.
- **Cache & Event Bus**: Redis (for caching and Redis Streams event broadcasting).
- **Migration & Tooling**: `goose` for schema tracking, structural CLI `entsaasctl` for maintenance.
- **Logging & Diagnostics**: `zerolog` structured logging, liveness and readiness endpoints.

### Frontend Client
- **Core UI**: Vue 3 featuring reactive Composition API.
- **State Management**: **Pinia** reactive store engines.
- **Build System**: **Vite** for lightning-fast hot module reloading.
- **Aesthetics & Design**: Clean HSL tailored styles with native Dark Mode support out-of-the-box.

---

## Core Framework Pillars

### 1. Multi-Tenant Isolation
Every HTTP request is intercepted and isolated by the tenant routing pipeline. Database connections are dynamically multiplexed based on organization claims, preventing cross-tenant data pollution.

### 2. Access Management & Security
Built on robust JSON Web Token (JWT) standards with seamless client-side Axios rotation interceptors. The Role-Based Access Control (RBAC) middleware guarantees users only reach routes suited for their roles:
- **Owner**: Complete organization ownership and billing control.
- **Admin**: Full administrative user invite and workspace management.
- **Member**: Core product usage and feature execution permissions.
- **Viewer**: Read-only observation states.

### 3. Declarative Catalog Billing
Rather than hardcoding tiers across multiple views and handler conditions, a single `config/billing.yaml` file drives active configurations:
- Plan versions sync dynamically to PostgreSQL on startup.
- Quotas (Project count, Team seats) and feature gates are evaluated in real-time.
- Multi-provider gateway support for Stripe and Paddle Billing out-of-the-box.

### 4. Gated Workspace AI Integration
A robust AI streaming engine supports streaming completions to modern client widgets. Includes dynamic plan gating (e.g. restrict AI prompt features to paid tiers) and support for OpenAI, Ollama, Azure, and vLLM providers.
