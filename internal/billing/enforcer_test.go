package billing_test

import (
	"context"
	"errors"
	"testing"

	"entsaas/internal/billing"
	"entsaas/internal/store"
)

// mockAppStore implements a mock for the AppStore interface methods used by the Enforcer.
type mockAppStore struct {
	store.AppStore // Embed to satisfy interface for methods we don't mock

	planSlug      string
	entitlements  map[string]any
	entitleErr    error
	projectsCount int
	projectsErr   error
	usersCount    int
	usersErr      error
}

func (m *mockAppStore) GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error) {
	return m.planSlug, m.entitlements, m.entitleErr
}

func (m *mockAppStore) CountProjects(ctx context.Context, orgID string) (int, error) {
	return m.projectsCount, m.projectsErr
}

func (m *mockAppStore) CountUsers(ctx context.Context, orgID string) (int, error) {
	return m.usersCount, m.usersErr
}

func TestEnforcer_CheckProjectLimit(t *testing.T) {
	t.Run("error getting entitlements", func(t *testing.T) {
		m := &mockAppStore{entitleErr: errors.New("db error")}
		e := billing.NewEnforcer(m)
		err := e.CheckProjectLimit(context.Background(), "org1")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("no limit configured (fail-open)", func(t *testing.T) {
		m := &mockAppStore{entitlements: map[string]any{}}
		e := billing.NewEnforcer(m)
		if err := e.CheckProjectLimit(context.Background(), "org1"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("unlimited (-1)", func(t *testing.T) {
		m := &mockAppStore{entitlements: map[string]any{string(billing.KeyMaxProjects): int64(-1)}}
		e := billing.NewEnforcer(m)
		if err := e.CheckProjectLimit(context.Background(), "org1"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("under limit", func(t *testing.T) {
		m := &mockAppStore{
			entitlements:  map[string]any{string(billing.KeyMaxProjects): int64(3)},
			projectsCount: 2,
		}
		e := billing.NewEnforcer(m)
		if err := e.CheckProjectLimit(context.Background(), "org1"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("at limit", func(t *testing.T) {
		m := &mockAppStore{
			entitlements:  map[string]any{string(billing.KeyMaxProjects): int64(3)},
			projectsCount: 3,
		}
		e := billing.NewEnforcer(m)
		err := e.CheckProjectLimit(context.Background(), "org1")
		var qErr *billing.QuotaExceededError
		if err == nil || !errors.As(err, &qErr) {
			t.Errorf("expected QuotaExceededError, got %v", err)
		}
	})
}

func TestEnforcer_CheckMemberLimit(t *testing.T) {
	t.Run("at limit", func(t *testing.T) {
		m := &mockAppStore{
			entitlements: map[string]any{string(billing.KeyMaxMembers): float64(5)}, // Test float64 conversion
			usersCount:   5,
		}
		e := billing.NewEnforcer(m)
		err := e.CheckMemberLimit(context.Background(), "org1")
		var qErr *billing.QuotaExceededError
		if err == nil || !errors.As(err, &qErr) {
			t.Errorf("expected QuotaExceededError, got %v", err)
		}
	})

	t.Run("under limit", func(t *testing.T) {
		m := &mockAppStore{
			entitlements: map[string]any{string(billing.KeyMaxMembers): int(5)}, // Test int conversion
			usersCount:   4,
		}
		e := billing.NewEnforcer(m)
		if err := e.CheckMemberLimit(context.Background(), "org1"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestEnforcer_CheckFeature(t *testing.T) {
	t.Run("feature missing", func(t *testing.T) {
		m := &mockAppStore{entitlements: map[string]any{}}
		e := billing.NewEnforcer(m)
		err := e.CheckFeature(context.Background(), "org1", "ai_assistant_enabled")
		var fErr *billing.FeatureUnavailableError
		if err == nil || !errors.As(err, &fErr) {
			t.Errorf("expected FeatureUnavailableError, got %v", err)
		}
	})

	t.Run("feature disabled", func(t *testing.T) {
		m := &mockAppStore{entitlements: map[string]any{"ai_assistant_enabled": false}}
		e := billing.NewEnforcer(m)
		err := e.CheckFeature(context.Background(), "org1", "ai_assistant_enabled")
		var fErr *billing.FeatureUnavailableError
		if err == nil || !errors.As(err, &fErr) {
			t.Errorf("expected FeatureUnavailableError, got %v", err)
		}
	})

	t.Run("feature enabled", func(t *testing.T) {
		m := &mockAppStore{entitlements: map[string]any{"ai_assistant_enabled": true}}
		e := billing.NewEnforcer(m)
		if err := e.CheckFeature(context.Background(), "org1", "ai_assistant_enabled"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
