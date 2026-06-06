package billing

// stripe.go — Stripe billing adapter for EntSaaS.
//
// To activate:
//   1. go get github.com/stripe/stripe-go/v82
//   2. Set STRIPE_SECRET_KEY + STRIPE_WEBHOOK_SECRET in your .env
//   3. Call billing.NewStripeProvider() from cmd/api/main.go
//   4. Update billing.New() factory to return this provider when keys are present.
//
// This file compiles unconditionally. The Stripe SDK is imported only when
// STRIPE_SECRET_KEY is non-empty (lazy import pattern via New()).

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// StripeProvider implements Provider using the Stripe API.
// Replaced stub HTTP calls with ProviderHTTPClient to keep zero-dependency.
type StripeProvider struct {
	secretKey     string
	webhookSecret string
	baseURL       string
	client        *ProviderHTTPClient
}

// NewStripeProvider constructs a StripeProvider from environment variables.
// Returns nil + error if the required env vars are absent.
func NewStripeProvider() (*StripeProvider, error) {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		return nil, errors.New("billing: STRIPE_SECRET_KEY is not set")
	}
	return &StripeProvider{
		secretKey:     key,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		baseURL:       "https://api.stripe.com/v1",
		client:        NewProviderHTTPClient(15*time.Second, 2),
	}, nil
}

// ── Provider interface implementation ────────────────────────────────────────

func (p *StripeProvider) ListPlans(_ context.Context) ([]*Plan, error) {
	// TODO: call stripe.Price.List() to retrieve active prices + products.
	// For now return the static catalog. Replace with SDK call once stripe-go is added.
	return NoopProvider{}.ListPlans(context.Background())
}

func (p *StripeProvider) GetSubscription(_ context.Context, orgID string) (*Subscription, error) {
	// TODO: look up the customer by metadata["org_id"] and retrieve their subscription.
	// Implementation sketch:
	//   customer, _ := stripe.Customer.Search({query: "metadata['org_id']:'"+orgID+"'"})
	//   sub, _ := stripe.Subscription.Get(customer.Subscriptions[0].ID)
	//   return mapStripeSubscription(sub, orgID), nil
	return NoopProvider{}.GetSubscription(context.Background(), orgID)
}

func (p *StripeProvider) CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*CheckoutSession, error) {
	priceID := ""
	if GlobalCatalog != nil {
		for _, plan := range GlobalCatalog.Plans {
			if plan.ID == planID {
				priceID = plan.StripePriceID
				break
			}
		}
	}
	if priceID == "" {
		return nil, fmt.Errorf("stripe: could not resolve price ID for plan %s", planID)
	}

	form := url.Values{}
	form.Add("mode", "subscription")
	form.Add("line_items[0][price]", priceID)
	form.Add("line_items[0][quantity]", "1")
	form.Add("success_url", successURL+"?session_id={CHECKOUT_SESSION_ID}")
	form.Add("cancel_url", cancelURL)
	form.Add("client_reference_id", orgID)
	form.Add("metadata[org_id]", orgID)

	req, err := p.client.NewFormRequest(ctx, http.MethodPost, p.baseURL+"/checkout/sessions", p.secretKey, form.Encode())
	if err != nil {
		return nil, err
	}

	body, _, err := p.client.Do(req, ProviderStripe)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("stripe: failed to parse checkout response: %w", err)
	}

	sessionID, _ := data["id"].(string)
	checkoutURL, _ := data["url"].(string)

	return &CheckoutSession{
		SessionID:   sessionID,
		CheckoutURL: checkoutURL,
	}, nil
}

func (p *StripeProvider) Name() ProviderName {
	return ProviderStripe
}

func (p *StripeProvider) CreatePortalSession(ctx context.Context, customerID string) (*CustomerPortalSession, error) {
	if customerID == "" {
		return nil, errors.New("stripe: customer ID required for portal session")
	}

	form := url.Values{}
	form.Add("customer", customerID)
	// Optional: add return_url

	req, err := p.client.NewFormRequest(ctx, http.MethodPost, p.baseURL+"/billing_portal/sessions", p.secretKey, form.Encode())
	if err != nil {
		return nil, err
	}

	body, _, err := p.client.Do(req, ProviderStripe)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("stripe: failed to parse portal response: %w", err)
	}

	urlStr, _ := data["url"].(string)

	return &CustomerPortalSession{
		PortalURL: urlStr,
	}, nil
}

func (p *StripeProvider) FetchBillingSummary(ctx context.Context, customerID string) (*ProviderBillingSummary, error) {
	return NoopProvider{}.FetchBillingSummary(ctx, customerID)
}

func (p *StripeProvider) UpdateSubscription(ctx context.Context, req SwitchRequest) error {
	return NoopProvider{}.UpdateSubscription(ctx, req)
}

func (p *StripeProvider) PreviewSubscriptionUpdate(ctx context.Context, subscriptionID, targetPriceID string) (*ProrationPreview, error) {
	return NoopProvider{}.PreviewSubscriptionUpdate(ctx, subscriptionID, targetPriceID)
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return NoopProvider{}.CancelSubscription(ctx, subscriptionID)
}

func (p *StripeProvider) VerifyWebhook(payload []byte, headers map[string]string, secret string) error {
	sig := headers["Stripe-Signature"]
	sec := secret
	if sec == "" {
		sec = p.webhookSecret
	}
	if sec == "" {
		return errors.New("stripe: webhook secret is not configured")
	}
	return p.verifyStripeSignature(payload, sig, sec)
}

// HandleWebhook verifies the Stripe-Signature header and dispatches the event.
func (p *StripeProvider) HandleWebhook(_ context.Context, event *WebhookEvent) (*WebhookResult, error) {
	// SEC-03: Reject unsigned webhooks — an empty secret means signature
	// verification would be skipped, allowing attackers to forge events.
	if p.webhookSecret == "" {
		return nil, errors.New("stripe: webhook secret is not configured — refusing unsigned event")
	}

	sig := event.Headers["Stripe-Signature"]
	if err := p.verifyStripeSignature(event.Raw, sig, p.webhookSecret); err != nil {
		return nil, fmt.Errorf("stripe: invalid webhook signature: %w", err)
	}

	// Parse the raw payload into structured data.
	var payload map[string]any
	if err := json.Unmarshal(event.Raw, &payload); err != nil {
		return nil, fmt.Errorf("stripe: failed to parse webhook body: %w", err)
	}

	eventType, _ := payload["type"].(string)
	result := &WebhookResult{EventType: eventType}

	// Extract the event object (data.object).
	data, _ := payload["data"].(map[string]any)
	obj, _ := data["object"].(map[string]any)
	if obj == nil {
		return result, nil
	}

	// Dispatch on event type and populate result fields.
	switch eventType {
	case "checkout.session.completed":
		// Provision subscription from checkout session.
		result.ProviderCustomerID, _ = obj["customer"].(string)
		result.ProviderSubscriptionID, _ = obj["subscription"].(string)
		result.Status = "active"
		if meta, ok := obj["metadata"].(map[string]any); ok {
			result.OrgID, _ = meta["org_id"].(string)
		}
		if result.OrgID == "" {
			result.OrgID, _ = obj["client_reference_id"].(string)
		}

	case "customer.subscription.updated":
		result.ProviderSubscriptionID, _ = obj["id"].(string)
		result.ProviderCustomerID, _ = obj["customer"].(string)
		result.Status, _ = obj["status"].(string)
		cancelEnd, _ := obj["cancel_at_period_end"].(bool)
		result.CancelAtPeriodEnd = cancelEnd
		if meta, ok := obj["metadata"].(map[string]any); ok {
			result.OrgID, _ = meta["org_id"].(string)
		}
		if start, ok := obj["current_period_start"].(float64); ok {
			t := time.Unix(int64(start), 0)
			result.CurrentPeriodStart = &t
		}
		if end, ok := obj["current_period_end"].(float64); ok {
			t := time.Unix(int64(end), 0)
			result.CurrentPeriodEnd = &t
		}

	case "customer.subscription.deleted":
		result.ProviderSubscriptionID, _ = obj["id"].(string)
		result.Status = "canceled"
		if meta, ok := obj["metadata"].(map[string]any); ok {
			result.OrgID, _ = meta["org_id"].(string)
		}

	case "invoice.payment_failed":
		result.ProviderSubscriptionID, _ = obj["subscription"].(string)
		result.ProviderCustomerID, _ = obj["customer"].(string)
		result.Status = "past_due"
	}

	return result, nil
}

// verifyStripeSignature validates the Stripe-Signature header (t=timestamp,v1=hmac).
func (p *StripeProvider) verifyStripeSignature(payload []byte, header string, secret string) error {
	parts := strings.Split(header, ",")
	var ts, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			ts = strings.TrimPrefix(part, "t=")
		}
		if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if ts == "" || sig == "" {
		return errors.New("malformed Stripe-Signature header")
	}

	// Tolerance: reject if older than 5 minutes.
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Now().Unix()-tsInt > 300 {
		return errors.New("stripe signature timestamp out of tolerance")
	}

	signed := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("stripe signature mismatch")
	}
	return nil
}
