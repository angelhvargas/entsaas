# System Overview

This section covers the core backend and frontend architecture of the EntSaaS framework.

```mermaid
graph LR
    %% High-level architecture and data flows
    subgraph Client Application [Vue 3 Client App]
        A[AppShell Vue Layout]
        B[Pinia Stores: auth.js, billing.js]
        C[Axios Client Interceptors]
    end
    
    subgraph API Router [Gin Web Server]
        D[JWT SessionAuth Middleware]
        E[RBAC RequireRole Middleware]
        F[Handlers: auth, billing, projects, ai]
    end
    
    subgraph Storage Layer [PostgreSQL & Redis]
        G[(Postgres DB Connection Pool)]
        H[(Redis Event Broker & Cache)]
    end
    
    C -->|JWT Auth Header| D
    D -->|Injects claims context| E
    E -->|Authorized request| F
    F -->|Multiplexed DB routing| G
    F -->|Event publish & TTL cache| H
```

---

## Key Component Layout

### 1. The Request Pipeline
1. The **Vue 3 Client** dispatches HTTP requests via a central Axios wrapper.
2. The request is intercepted by the **`SessionAuth()`** middleware, which extracts and validates JWT access claims.
3. The **`RequireRole()`** middleware asserts the user's role satisfies endpoint rules (e.g. only Admins can create team invitations).
4. Handlers fetch DB pools via the **Tenant Router** and return structured JSON or stream Server-Sent Events (SSE).

### 2. Database Connection Pooling
EntSaaS leverages `pgx` with integrated connection pools. Every database action is executed under org-isolated contexts, ensuring robust multi-tenant data segmentation.

### 3. Asynchronous Events & Caching
Redis Streams are used to decouple long-running operations (e.g., triggering invitation emails or executing billing reconciliations) from the core request-response thread, while the store cache handles high-performance TTL data retrieval.
