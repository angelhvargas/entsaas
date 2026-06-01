package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entsaas/internal/billing"
	"entsaas/internal/handlers"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

type mockProjectsStore struct {
	store.AppStore
	projectsCount int
	entitlements  map[string]any
}

func (m *mockProjectsStore) GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error) {
	return "free", m.entitlements, nil
}

func (m *mockProjectsStore) CountProjects(ctx context.Context, orgID string) (int, error) {
	return m.projectsCount, nil
}

func (m *mockProjectsStore) CreateProject(ctx context.Context, orgID, name string) (*store.Project, error) {
	return &store.Project{ID: "proj1", Name: name, OrgID: orgID}, nil
}

func (m *mockProjectsStore) LogAuditEvent(ctx context.Context, orgID *string, actorID, action, resourceType, resourceID string, metadata map[string]any) error {
	return nil
}

func TestProjectHandler_CreateProject_Quota(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("under quota", func(t *testing.T) {
		mStore := &mockProjectsStore{
			entitlements:  map[string]any{string(billing.KeyMaxProjects): int64(3)},
			projectsCount: 2,
		}
		h := handlers.NewProjectHandler(mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"new proj"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		if w.Code != http.StatusCreated {
			t.Errorf("expected 201 Created, got %d", w.Code)
		}
	})

	t.Run("over quota", func(t *testing.T) {
		mStore := &mockProjectsStore{
			entitlements:  map[string]any{string(billing.KeyMaxProjects): int64(3)},
			projectsCount: 3, // At limit, should block
		}
		h := handlers.NewProjectHandler(mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"new proj"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402 Payment Required, got %d", w.Code)
		}
	})
}
