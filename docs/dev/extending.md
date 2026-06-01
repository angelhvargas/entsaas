# Extending EntSaaS

This guide covers the most common extension tasks: adding API endpoints, Vue views, Pinia stores, and database migrations.

## Adding a New API Endpoint

### 1. Define the store method (if needed)

Add the method to the `AppStore` interface in [internal/store/interfaces.go](../../internal/store/interfaces.go):

```go
// In the AppStore interface
GetWidgetsByProject(ctx context.Context, projectID string) ([]*Widget, error)
```

### 2. Implement it in Postgres

Add the implementation in [internal/store/postgres.go](../../internal/store/postgres.go):

```go
func (s *PostgresStore) GetWidgetsByProject(ctx context.Context, projectID string) ([]*Widget, error) {
    rows, err := s.pool.Query(ctx,
        `SELECT id, name, project_id, created_at FROM widgets WHERE project_id = $1`, projectID)
    // ... scan rows
}
```

### 3. Create the handler

Create `internal/handlers/widgets.go`:

```go
type WidgetHandler struct {
    store store.AppStore
}

func NewWidgetHandler(s store.AppStore) *WidgetHandler {
    return &WidgetHandler{store: s}
}

func (h *WidgetHandler) List(c *gin.Context) {
    projectID := c.Param("project_id")
    widgets, err := h.store.GetWidgetsByProject(c.Request.Context(), projectID)
    // ... return JSON
}
```

### 4. Register the route

Add to the appropriate surface in `internal/api/`:

```go
// In routes_app.go
widgetHandler := handlers.NewWidgetHandler(deps.Store)
app.GET("/projects/:project_id/widgets", widgetHandler.List)
```

### 5. Create a migration (if needed)

```sql
-- internal/migrations/postgres/002_widgets.sql
-- +goose Up
CREATE TABLE widgets (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS widgets;
```

Run: `make migrate`

---

## Adding a New Vue View

### 1. Create the view component

```bash
web/src/views/WidgetsView.vue
```

### 2. Add the route

In `web/src/router/index.js`, add inside the AppShell children:

```js
{
    path: 'projects/:projectId/widgets',
    name: 'widgets',
    component: () => import('@/views/WidgetsView.vue'),
}
```

### 3. Add navigation (optional)

Update `AppShell.vue` nav items:

```js
{ name: 'Widgets', icon: Blocks, to: '/widgets' }
```

---

## Adding a New Pinia Store

Create `web/src/stores/widgets.js`:

```js
import { defineStore } from 'pinia'
import { ref } from 'vue'
import client from '@/api/client'

export const useWidgetsStore = defineStore('widgets', () => {
    const items = ref([])
    const loading = ref(false)

    async function fetch(projectId) {
        loading.value = true
        try {
            const { data } = await client.get(`/projects/${projectId}/widgets`)
            items.value = data.widgets || []
        } finally {
            loading.value = false
        }
    }

    return { items, loading, fetch }
})
```

---

## Adding AI Capabilities

### Custom system prompt

Pass a system message as the first message in the `messages` array:

```json
{
    "messages": [
        { "role": "system", "content": "You are a helpful assistant specialized in..." },
        { "role": "user", "content": "How do I..." }
    ]
}
```

### Using a different model

Pass the `model` field to override the default:

```json
{
    "messages": [...],
    "model": "gpt-4o-mini"
}
```

---

## Development Workflow

```bash
# After making backend changes:
go build -buildvcs=false ./...   # Verify compilation
go test -short ./...              # Run unit tests
go vet ./...                      # Run linter

# After making frontend changes:
cd web && npm run dev             # Hot reload
cd web && npx vitest run          # Run tests

# After adding a migration:
make migrate                      # Apply to dev DB
```
