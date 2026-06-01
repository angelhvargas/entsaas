---
layout: home

hero:
  name: "EntSaaS"
  text: "Production Go + Vue 3 Multi-Tenant SaaS Framework"
  tagline: "Build securely, scale horizontally, and manage dynamically. A robust multi-tenant core pre-wired with JWT, role-based RBAC, declarative catalog billing, streaming AI workspace chat, and comprehensive operations CLI."
  image:
    light: "/entsaas-logo-positive.svg"
    dark: "/entsaas-logo-negative.svg"
    alt: "EntSaaS Framework Logo"
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quickstart
    - theme: alt
      text: Architecture Deep-Dives
      link: /architecture/

features:
  - title: 🚀 Multi-Tenant Segregation
    details: Segment database routing dynamically or isolate schemas. Multi-tenant workspace segregation pre-wired for seamless customer safety.
    link: /architecture/multi-tenancy
    linkText: Multi-Tenancy Router →
  - title: 🛡️ Security & RBAC
    details: Complete JWT authentication token rotation, refresh loops, and strict Role-Based Access Control (Owner, Admin, Member, Viewer).
    link: /architecture/auth
    linkText: Access Security →
  - title: 💳 Declarative Catalog Billing
    details: Synchronize billing.yaml catalog limits dynamically. Pre-wired for Stripe/Paddle checkout portals, dunning events, and webhook synchronization.
    link: /architecture/billing
    linkText: Billing Catalog →
  - title: 📡 SSE Streaming AI Chat
    details: Real-time Server-Sent Events (SSE) AI completions pre-wired with OpenAI, Ollama, Azure, and vLLM providers under subscription plan gates.
    link: /architecture/ai-integration
    linkText: AI Engineering →
  - title: ⚙️ Developer CLI
    details: Powerful CLI operations tool `entsaasctl` to seed databases, run migrations, and manage team invitation lifecycle safely.
    link: /guide/cli
    linkText: CLI Administration →
  - title: 📈 Operational Readiness
    details: Out-of-the-box zerolog structured logging, Prometheus metric gates, request trace IDs, and Kubernetes readiness/liveness endpoints.
    link: /architecture/
    linkText: Observability Engine →

---

## Engineered for scale and developer speed

EntSaaS is a **premium multi-tenant framework boilerplate** written in high-performance Go (Gin router) and modern Vue 3 (Pinia state managers, Vite, Tailwind CSS compatible). It solves the complex foundational pipelines of SaaS architecture so you can focus entirely on your core product logic.

### Framework Features Out-of-the-Box:

#### 1. Multi-Tenant Routing Layer
Dynamic database tenant resolution segmenting operational queries without manual cross-tenant data leaks. Ready for horizontal growth.

#### 2. Advanced Auth Lifecycle
JWT access token rotations, expired request buffers in frontend API clients, role-based route constraints, secure welcome and password recovery SMTP pipelines.

#### 3. Quota-Driven Billing Engine
A single declarative `billing.yaml` file defines your plan catalog and pricing tiers. EntSaaS handles the startup sync, live member limits, and webhook sync from payment gateways.

#### 4. Streaming SSE Workspace AI
A multi-provider AI completion engine supporting server-sent event (SSE) streaming chats directly to interactive front-end chat widgets under subscription feature gates.
