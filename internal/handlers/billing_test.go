package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entsaas/internal/billing"
	"entsaas/internal/handlers"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

// mockAppStore
type mockAppStore struct {
	store.AppStore
	planSlug     string
	entitlements map[string]any
	entitleErr   error
}

func (m *mockAppStore) GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error) {
	return m.planSlug, m.entitlements, m.entitleErr
}

// mockBillingProvider
type mockBillingProvider struct {
	billing.Provider
	checkoutSession *billing.CheckoutSession
	checkoutErr     error
}

func (m *mockBillingProvider) CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*billing.CheckoutSession, error) {
	return m.checkoutSession, m.checkoutErr
}

func TestBillingHandler_GetPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mStore := &mockAppStore{
			planSlug:     "pro",
			entitlements: map[string]any{"ai_assistant": true},
		}
		h := handlers.NewBillingHandler(nil, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodGet, "/billing/plan", nil)

		h.GetPlan(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"plan_slug":"pro"`) {
			t.Errorf("expected body to contain plan_slug pro, got %s", w.Body.String())
		}
	})

	t.Run("error", func(t *testing.T) {
		mStore := &mockAppStore{
			entitleErr: errors.New("db error"),
		}
		h := handlers.NewBillingHandler(nil, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodGet, "/billing/plan", nil)

		h.GetPlan(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError, got %d", w.Code)
		}
	})
}

func TestBillingHandler_CreateCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mProvider := &mockBillingProvider{
			checkoutSession: &billing.CheckoutSession{CheckoutURL: "https://checkout.stripe.com/xyz"},
		}
		h := handlers.NewBillingHandler(mProvider, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/billing/checkout", strings.NewReader(`{"plan_id":"pro"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.CreateCheckout(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `https://checkout.stripe.com/xyz`) {
			t.Errorf("expected body to contain checkout URL, got %s", w.Body.String())
		}
	})

	t.Run("provider error", func(t *testing.T) {
		mProvider := &mockBillingProvider{
			checkoutErr: errors.New("provider error"),
		}
		h := handlers.NewBillingHandler(mProvider, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/billing/checkout", strings.NewReader(`{"plan_id":"pro"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.CreateCheckout(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}
