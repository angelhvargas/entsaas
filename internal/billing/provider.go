package billing

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ── Domain Types ─────────────────────────────────────────────────────────────

// Plan represents a billing plan available in the system.
type Plan struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	PriceMonthly float64  `json:"price_monthly"`
	PriceYearly  float64  `json:"price_yearly"`
	Currency     string   `json:"currency"`
	Features     []string `json:"features"`
	IsActive     bool     `json:"is_active"`
}

// Subscription represents an org's current billing subscription.
type Subscription struct {
	ID                 string     `json:"id"`
	OrgID              string     `json:"org_id"`
	PlanID             string     `json:"plan_id"`
	Status             string     `json:"status"` // active, trialing, past_due, canceled, paused
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	CancelAtPeriodEnd  bool       `json:"cancel_at_period_end"`
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

// ── Multi-Provider Billing Connector Types ───────────────────────────────────

// ProviderName identifies the active billing provider.
type ProviderName string

const (
	ProviderStripe ProviderName = "stripe"
	ProviderPaddle ProviderName = "paddle"
	ProviderNoop   ProviderName = "noop"
)

// SwitchRequest is the input for updating (switching) an active subscription.
type SwitchRequest struct {
	SubscriptionID string            // provider subscription identifier
	TargetPriceID  string            // provider price identifier for the target plan
	ProrationMode  string            // "prorated_immediately" | "do_not_bill"
	Metadata       map[string]string // EntSaaS metadata for the switch event
}

// ProrationPreview is the normalized result of a subscription update preview.
// Amounts are in the subscription currency's smallest unit (e.g. cents for EUR/GBP/USD).
type ProrationPreview struct {
	// CreditAmountCents is the credit for unused time on the current plan (always >= 0).
	CreditAmountCents int `json:"credit_amount_cents"`
	// ChargeAmountCents is the charge for the new plan's prorated period (always >= 0).
	ChargeAmountCents int `json:"charge_amount_cents"`
	// NetAmountCents: positive = customer is charged now, negative = customer receives credit.
	NetAmountCents int `json:"net_amount_cents"`
	// Currency is the ISO 4217 lowercase currency code (e.g. "eur", "usd").
	Currency string `json:"currency"`
	// EffectiveDate is when the switch takes effect (RFC3339).
	EffectiveDate string `json:"effective_date"`
	// NextBillingDate is the next renewal date after the switch (RFC3339).
	NextBillingDate string `json:"next_billing_date"`
	// NextBillingAmountCents is the recurring amount from the next billing cycle.
	NextBillingAmountCents int `json:"next_billing_amount_cents"`
}

// ProviderBillingSummary is the normalized billing summary from a provider.
type ProviderBillingSummary struct {
	PaymentMethod *NormalizedPaymentMethod `json:"payment_method,omitempty"`
	Invoices      []NormalizedInvoice      `json:"invoices"`
	RenewalDate   *time.Time               `json:"renewal_date,omitempty"`
}

// NormalizedPaymentMethod is a provider-agnostic payment method summary.
type NormalizedPaymentMethod struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

// NormalizedInvoice is a provider-agnostic invoice/transaction record.
type NormalizedInvoice struct {
	ID            string `json:"id"`
	Status        string `json:"status"` // paid, open, past_due, void
	AmountCents   int    `json:"amount_cents"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"created_at"`
	HostedURL     string `json:"hosted_url,omitempty"`
	PDFUrl        string `json:"pdf_url,omitempty"`
	Description   string `json:"description,omitempty"`
	InvoiceNumber string `json:"invoice_number,omitempty"`
	IsPlanChange  bool   `json:"is_plan_change,omitempty"`
	PlanChangeCtx string `json:"plan_change_ctx,omitempty"`
}

// ── Provider Advisory State ──────────────────────────────────────────────────

// AdvisoryStatus represents a provider-reported payment/billing condition.
type AdvisoryStatus string

const (
	AdvisoryPaymentOK             AdvisoryStatus = "payment_ok"
	AdvisoryPaymentWarning        AdvisoryStatus = "payment_warning"
	AdvisoryPaymentPastDue        AdvisoryStatus = "payment_past_due"
	AdvisoryPaymentMethodExpiring AdvisoryStatus = "payment_method_expiring"
	AdvisoryCancellationPending   AdvisoryStatus = "provider_cancellation_pending"
	AdvisoryTrialExpiring         AdvisoryStatus = "trial_expiring"

	// AdvisoryCheckoutPending indicates the user completed a hosted checkout
	// flow but the definitive provider paid/active signal has not yet been
	// received.
	AdvisoryCheckoutPending AdvisoryStatus = "checkout_pending"
)

// ValidAdvisoryStatuses is the set of accepted advisory status values.
var ValidAdvisoryStatuses = map[AdvisoryStatus]bool{
	AdvisoryPaymentOK:             true,
	AdvisoryPaymentWarning:        true,
	AdvisoryPaymentPastDue:        true,
	AdvisoryPaymentMethodExpiring: true,
	AdvisoryCancellationPending:   true,
	AdvisoryTrialExpiring:         true,
	AdvisoryCheckoutPending:       true,
}

// IsValidAdvisoryStatus reports whether s is a recognized advisory status.
func IsValidAdvisoryStatus(s AdvisoryStatus) bool {
	return ValidAdvisoryStatuses[s]
}

// ── Errors ───────────────────────────────────────────────────────────────────

var (
	ErrPortalNotSupported    = errors.New("billing: portal not supported by provider")
	ErrProviderNotConfigured = errors.New("billing: provider not configured")
	ErrUnknownProvider       = errors.New("billing: unknown provider name")
	ErrProviderAPIFailed     = errors.New("billing: provider API call failed")
)

// ProviderAPIError is a structured error from a provider API call.
// Contains safe information for logging/surfacing without leaking provider internals.
type ProviderAPIError struct {
	Provider   ProviderName // which provider returned the error
	StatusCode int          // HTTP status code (0 if no HTTP response)
	Message    string       // safe, user-facing message
	Underlying error        // original error (not exposed to end users)
}

func (e *ProviderAPIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("billing: %s API error (HTTP %d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("billing: %s API error: %s", e.Provider, e.Message)
}

func (e *ProviderAPIError) Unwrap() error { return e.Underlying }

// providerError creates a ProviderAPIError for the given provider.
func providerError(provider ProviderName, statusCode int, message string, underlying error) *ProviderAPIError {
	return &ProviderAPIError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Underlying: underlying,
	}
}

// ── Provider Interface ───────────────────────────────────────────────────────

// Provider defines the billing operations that any payment processor must implement.
type Provider interface {
	// Name returns the provider identifier.
	Name() ProviderName

	// ListPlans returns all available plans from the billing provider.
	ListPlans(ctx context.Context) ([]*Plan, error)

	// GetSubscription returns the current subscription for an org.
	GetSubscription(ctx context.Context, orgID string) (*Subscription, error)

	// CreateCheckoutSession initiates an upgrade flow.
	// successURL and cancelURL are the redirect targets after checkout.
	CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*CheckoutSession, error)

	// CreatePortalSession returns a billing portal URL for managing the subscription.
	CreatePortalSession(ctx context.Context, customerID string) (*CustomerPortalSession, error)

	// HandleWebhook processes an inbound webhook and returns structured result data.
	// The caller (billing handler) is responsible for persisting subscription state.
	HandleWebhook(ctx context.Context, event *WebhookEvent) (*WebhookResult, error)

	// FetchBillingSummary retrieves provider-backed billing information
	// (invoices, payment method, renewal date) for a customer.
	FetchBillingSummary(ctx context.Context, customerID string) (*ProviderBillingSummary, error)

	// UpdateSubscription changes an active subscription to a new price/plan.
	UpdateSubscription(ctx context.Context, req SwitchRequest) error

	// PreviewSubscriptionUpdate returns proration amounts for a planned switch
	// without committing the change. Safe to call multiple times.
	PreviewSubscriptionUpdate(ctx context.Context, subscriptionID, targetPriceID string) (*ProrationPreview, error)

	// CancelSubscription schedules cancellation at next billing period end.
	CancelSubscription(ctx context.Context, subscriptionID string) error

	// VerifyWebhook verifies the webhook payload authenticity using signature verification.
	VerifyWebhook(payload []byte, headers map[string]string, secret string) error
}

// ── No-op Provider (default) ──────────────────────────────────────────────────

// NoopProvider is a safe default that returns empty/error responses.
// Use in development before a real billing provider is configured.
type NoopProvider struct{}

func (NoopProvider) Name() ProviderName {
	return ProviderNoop
}

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

func (NoopProvider) FetchBillingSummary(_ context.Context, customerID string) (*ProviderBillingSummary, error) {
	now := time.Now()
	return &ProviderBillingSummary{
		PaymentMethod: &NormalizedPaymentMethod{
			Brand:    "Visa",
			Last4:    "4242",
			ExpMonth: 12,
			ExpYear:  2030,
		},
		Invoices: []NormalizedInvoice{
			{
				ID:            "noop-inv-1",
				Status:        "paid",
				AmountCents:   4900,
				Currency:      "usd",
				CreatedAt:     now.AddDate(0, 0, -30).Format(time.RFC3339),
				Description:   "Pro Plan - Monthly",
				InvoiceNumber: "INV-0001",
			},
		},
		RenewalDate: &now,
	}, nil
}

func (NoopProvider) UpdateSubscription(_ context.Context, _ SwitchRequest) error {
	return nil
}

func (NoopProvider) PreviewSubscriptionUpdate(_ context.Context, _, _ string) (*ProrationPreview, error) {
	now := time.Now()
	return &ProrationPreview{
		CreditAmountCents:      1000,
		ChargeAmountCents:      4900,
		NetAmountCents:         3900,
		Currency:               "usd",
		EffectiveDate:          now.Format(time.RFC3339),
		NextBillingDate:        now.AddDate(0, 1, 0).Format(time.RFC3339),
		NextBillingAmountCents: 4900,
	}, nil
}

func (NoopProvider) CancelSubscription(_ context.Context, _ string) error {
	return nil
}

func (NoopProvider) VerifyWebhook(_ []byte, _ map[string]string, _ string) error {
	return nil
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
