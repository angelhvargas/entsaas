package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	subscription *store.Subscription
	subErr       error
}

func (m *mockAppStore) GetEffectiveEntitlements(ctx context.Context, orgID string) (string, map[string]any, error) {
	return m.planSlug, m.entitlements, m.entitleErr
}

func (m *mockAppStore) GetSubscriptionByOrgID(ctx context.Context, orgID string) (*store.Subscription, error) {
	return m.subscription, m.subErr
}

// mockBillingProvider
type mockBillingProvider struct {
	billing.Provider
	checkoutSession *billing.CheckoutSession
	checkoutErr     error
	portalSession   *billing.CustomerPortalSession
	portalErr       error
	summary         *billing.ProviderBillingSummary
	summaryErr      error
}

func (m *mockBillingProvider) CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*billing.CheckoutSession, error) {
	return m.checkoutSession, m.checkoutErr
}

func (m *mockBillingProvider) CreatePortalSession(ctx context.Context, customerID string) (*billing.CustomerPortalSession, error) {
	return m.portalSession, m.portalErr
}

func (m *mockBillingProvider) FetchBillingSummary(ctx context.Context, customerID string) (*billing.ProviderBillingSummary, error) {
	return m.summary, m.summaryErr
}

func (m *mockBillingProvider) Name() billing.ProviderName {
	return billing.ProviderNoop
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

func TestBillingHandler_CreatePortal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mStore := &mockAppStore{
			subscription: &store.Subscription{
				ProviderCustomerID:     "cus_test",
				ProviderSubscriptionID: "sub_test",
			},
		}
		mProvider := &mockBillingProvider{
			portalSession: &billing.CustomerPortalSession{PortalURL: "https://billing.stripe.com/p/123"},
		}
		h := handlers.NewBillingHandler(mProvider, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/billing/portal", nil)

		h.CreatePortal(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `https://billing.stripe.com/p/123`) {
			t.Errorf("expected body to contain portal URL, got %s", w.Body.String())
		}
	})

	t.Run("subscription query error", func(t *testing.T) {
		mStore := &mockAppStore{
			subErr: errors.New("db error"),
		}
		h := handlers.NewBillingHandler(nil, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodPost, "/billing/portal", nil)

		h.CreatePortal(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestBillingHandler_GetBillingSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success provider backed", func(t *testing.T) {
		mStore := &mockAppStore{
			subscription: &store.Subscription{
				PlanSlug:               "pro",
				Status:                 "active",
				ProviderSubscriptionID: "sub_stripe_123",
				ProviderCustomerID:     "cus_stripe_123",
				CreatedAt:              time.Now(),
			},
		}

		now := time.Now()
		mProvider := &mockBillingProvider{
			summary: &billing.ProviderBillingSummary{
				PaymentMethod: &billing.NormalizedPaymentMethod{
					Brand: "Visa",
					Last4: "1111",
				},
				Invoices: []billing.NormalizedInvoice{
					{ID: "inv_1", Status: "paid", AmountCents: 4900, Currency: "usd"},
				},
				RenewalDate: &now,
			},
		}

		h := handlers.NewBillingHandler(mProvider, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodGet, "/billing/summary", nil)

		h.GetBillingSummary(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, `"plan_slug":"pro"`) || !strings.Contains(body, `"brand":"Visa"`) || !strings.Contains(body, `"last4":"1111"`) {
			t.Errorf("expected body to contain sub and pm details, got %s", body)
		}
	})

	t.Run("subscription not found", func(t *testing.T) {
		mStore := &mockAppStore{
			subscription: nil,
		}
		h := handlers.NewBillingHandler(nil, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodGet, "/billing/summary", nil)

		h.GetBillingSummary(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("query error", func(t *testing.T) {
		mStore := &mockAppStore{
			subErr: errors.New("db error"),
		}
		h := handlers.NewBillingHandler(nil, mStore)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("org_id", "org1")
		c.Request, _ = http.NewRequest(http.MethodGet, "/billing/summary", nil)

		h.GetBillingSummary(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestBillingHandler_GetInvoices_InvalidStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Since mockAppStore cannot be type asserted to *store.PostgresStore, it should return 500.
	h := handlers.NewBillingHandler(nil, &mockAppStore{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("org_id", "org1")
	c.Request, _ = http.NewRequest(http.MethodGet, "/billing/invoices", nil)

	h.GetInvoices(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid store assertion, got %d", w.Code)
	}
}

func TestBillingHandler_GetInvoice_InvalidStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handlers.NewBillingHandler(nil, &mockAppStore{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("org_id", "org1")
	c.Params = []gin.Param{{Key: "id", Value: "inv_123"}}
	c.Request, _ = http.NewRequest(http.MethodGet, "/billing/invoices/inv_123", nil)

	h.GetInvoice(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid store assertion, got %d", w.Code)
	}
}
