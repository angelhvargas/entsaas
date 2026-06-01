package billing

import (
	"context"
	"time"
)

// ── Domain Types ─────────────────────────────────────────────────────────────

// Plan represents a billing plan available in the system.
type Plan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceMonthly float64 `json:"price_monthly"`
	PriceYearly  float64 `json:"price_yearly"`
	Currency     string  `json:"currency"`
	Features     []string `json:"features"`
	IsActive     bool    `json:"is_active"`
}

// Subscription represents an org's current billing subscription.
type Subscription struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"org_id"`
	PlanID          string     `json:"plan_id"`
	Status          string     `json:"status"` // active, trialing, past_due, canceled, paused
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
	TrialEndAt         *time.Time `json:"trial_end_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CheckoutSession is returned when a user initiates a plan upgrade.
type CheckoutSession struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"` // redirect the user here
}

// CustomerPortalSession is returned for billing portal access.
type CustomerPortalSession struct {
	PortalURL string `json:"portal_url"`
}

// WebhookEvent wraps a raw webhook payload for processing.
type WebhookEvent struct {
	Type    string            `json:"type"`
	Payload map[string]any    `json:"payload"`
	Raw     []byte            `json:"-"` // raw body for signature verification
	Headers map[string]string `json:"-"`
}

// WebhookResult is returned by HandleWebhook with parsed event data.
// The billing handler uses this to persist subscription state.
type WebhookResult struct {
	EventType              string     `json:"event_type"`
	OrgID                  string     `json:"org_id,omitempty"`
	PlanSlug               string     `json:"plan_slug,omitempty"`
	ProviderSubscriptionID string     `json:"provider_subscription_id,omitempty"`
	ProviderCustomerID     string     `json:"provider_customer_id,omitempty"`
	Status                 string     `json:"status,omitempty"` // active, canceled, past_due, etc.
	CurrentPeriodStart     *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd       *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end,omitempty"`
}

// ── Provider Interface ───────────────────────────────────────────────────────

// Provider defines the billing operations that any payment processor must implement.
// Swap implementations (Stripe, Paddle, LemonSqueezy) without changing handlers.
type Provider interface {
	// ListPlans returns all available plans from the billing provider.
	ListPlans(ctx context.Context) ([]*Plan, error)

	// GetSubscription returns the current subscription for an org.
	GetSubscription(ctx context.Context, orgID string) (*Subscription, error)

	// CreateCheckoutSession initiates an upgrade flow.
	// successURL and cancelURL are the redirect targets after checkout.
	CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*CheckoutSession, error)

	// CreatePortalSession returns a billing portal URL for managing the subscription.
	CreatePortalSession(ctx context.Context, orgID string) (*CustomerPortalSession, error)

	// HandleWebhook processes an inbound webhook and returns structured result data.
	// The caller (billing handler) is responsible for persisting subscription state.
	HandleWebhook(ctx context.Context, event *WebhookEvent) (*WebhookResult, error)
}

// ── No-op Provider (default) ──────────────────────────────────────────────────

// NoopProvider is a safe default that returns empty/error responses.
// Use in development before a real billing provider is configured.
type NoopProvider struct{}

func (NoopProvider) ListPlans(_ context.Context) ([]*Plan, error) {
	return []*Plan{
		{ID: "free", Name: "Free", Description: "Up to 3 projects", PriceMonthly: 0, Currency: "usd", IsActive: true},
		{ID: "pro", Name: "Pro", Description: "Unlimited projects + AI", PriceMonthly: 49, Currency: "usd", IsActive: true},
		{ID: "enterprise", Name: "Enterprise", Description: "Custom limits + SLA", PriceMonthly: 0, Currency: "usd", IsActive: true},
	}, nil
}

func (NoopProvider) GetSubscription(_ context.Context, orgID string) (*Subscription, error) {
	return &Subscription{
		ID:                 "noop-sub",
		OrgID:              orgID,
		PlanID:             "free",
		Status:             "active",
		CurrentPeriodStart: time.Now().AddDate(0, -1, 0),
		CurrentPeriodEnd:   time.Now().AddDate(0, 0, 0),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

func (NoopProvider) CreateCheckoutSession(_ context.Context, _, _, _, cancelURL string) (*CheckoutSession, error) {
	return &CheckoutSession{
		SessionID:   "noop",
		CheckoutURL: cancelURL + "?billing=unavailable",
	}, nil
}

func (NoopProvider) CreatePortalSession(_ context.Context, _ string) (*CustomerPortalSession, error) {
	return &CustomerPortalSession{PortalURL: "#"}, nil
}

func (NoopProvider) HandleWebhook(_ context.Context, event *WebhookEvent) (*WebhookResult, error) {
	return &WebhookResult{EventType: event.Type}, nil
}

// ── Factory ─────────────────────────────────────────────────────────────────

// New returns the configured billing provider based on available env vars.
//
// Priority:
//   1. Stripe  — when STRIPE_SECRET_KEY is set
//   2. Paddle  — when PADDLE_API_KEY is set
//   3. NoopProvider — safe default for local dev
//
// The selected provider is logged at startup so it's never a surprise.
func New() Provider {
	if p, err := NewStripeProvider(); err == nil {
		return p
	}
	if p, err := NewPaddleProvider(); err == nil {
		return p
	}
	return NoopProvider{}
}
