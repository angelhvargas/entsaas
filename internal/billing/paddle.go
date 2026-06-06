package billing

// paddle.go — Paddle Billing adapter for EntSaaS.
//
// Paddle is Merchant of Record — it handles VAT/tax collection automatically.
//
// To activate:
//   1. go get github.com/PaddleHQ/paddle-go-sdk (or use raw HTTP)
//   2. Set PADDLE_API_KEY + PADDLE_WEBHOOK_SECRET in your .env
//   3. Call billing.NewPaddleProvider() from cmd/api/main.go
//   4. Update billing.New() factory to return this provider.
//
// Paddle API reference: https://developer.paddle.com/api-reference

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// PaddleProvider implements Provider using the Paddle Billing API.
type PaddleProvider struct {
	apiKey        string
	webhookSecret string
	baseURL       string
	sandbox       bool
	client        *ProviderHTTPClient
}

// NewPaddleProvider constructs a PaddleProvider from environment variables.
// Set PADDLE_SANDBOX=true for sandbox mode (non-production checkouts).
func NewPaddleProvider() (*PaddleProvider, error) {
	key := os.Getenv("PADDLE_API_KEY")
	if key == "" {
		return nil, errors.New("billing: PADDLE_API_KEY is not set")
	}
	sandbox := os.Getenv("PADDLE_SANDBOX") == "true"
	baseURL := "https://api.paddle.com"
	if sandbox {
		baseURL = "https://sandbox-api.paddle.com"
	}
	return &PaddleProvider{
		apiKey:        key,
		webhookSecret: os.Getenv("PADDLE_WEBHOOK_SECRET"),
		baseURL:       baseURL,
		sandbox:       sandbox,
		client:        NewProviderHTTPClient(15*time.Second, 2),
	}, nil
}

// ── Provider interface implementation ────────────────────────────────────────

func (p *PaddleProvider) ListPlans(_ context.Context) ([]*Plan, error) {
	// TODO: GET /products?status=active
	//   then GET /prices?product_id=... to attach pricing.
	// Cascade the plan_versions.provider_price_id for each tier.
	return NoopProvider{}.ListPlans(context.Background())
}

func (p *PaddleProvider) GetSubscription(ctx context.Context, orgID string) (*Subscription, error) {
	// TODO: GET /subscriptions?custom_data[org_id]=orgID
	// Paddle subscriptions carry custom_data set at checkout time.
	_ = orgID
	return NoopProvider{}.GetSubscription(ctx, orgID)
}

func (p *PaddleProvider) CreateCheckoutSession(ctx context.Context, orgID, planID, successURL, cancelURL string) (*CheckoutSession, error) {
	// Paddle uses client-side overlay checkout triggered by paddle.js.
	// The "session" here is a Paddle transaction (POST /transactions).
	
	// We lookup the provider price ID from the global catalog using the internal planID
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
		return nil, fmt.Errorf("paddle: could not resolve price ID for plan %s", planID)
	}

	body := map[string]any{
		"items": []map[string]any{{"price_id": priceID, "quantity": 1}},
		"custom_data": map[string]any{"org_id": orgID},
	}
	// Note: Paddle handles success URLs client side via paddle.js, 
	// but we can pass it if creating a generic transaction link.
	
	resp, err := p.post(ctx, "/transactions", body)
	if err != nil {
		return nil, err
	}
	
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		return nil, errors.New("paddle: invalid response missing data")
	}
	
	txnID, _ := data["id"].(string)
	checkoutURL, _ := data["checkout_url"].(string) // If checkout_url is provided by API
	if checkoutURL == "" {
		checkoutURL = fmt.Sprintf("https://checkout.paddle.com/checkout/transactions/%s", txnID)
	}

	return &CheckoutSession{
		SessionID:   txnID,
		CheckoutURL: checkoutURL,
	}, nil
}

func (p *PaddleProvider) Name() ProviderName {
	return ProviderPaddle
}

func (p *PaddleProvider) CreatePortalSession(ctx context.Context, customerID string) (*CustomerPortalSession, error) {
	if customerID == "" {
		return nil, errors.New("paddle: customer ID required for portal session")
	}
	path := fmt.Sprintf("/customers/%s/portal-sessions", customerID)
	resp, err := p.post(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		return nil, errors.New("paddle: invalid portal response")
	}
	
	urls, _ := data["urls"].(map[string]any)
	if urls == nil {
		return nil, errors.New("paddle: no portal urls returned")
	}
	
	portalURL, _ := urls["general"].(map[string]any)
	urlStr, _ := portalURL["url"].(string)
	
	if urlStr == "" {
		return nil, errors.New("paddle: empty portal url returned")
	}

	return &CustomerPortalSession{
		PortalURL: urlStr,
	}, nil
}

func (p *PaddleProvider) FetchBillingSummary(ctx context.Context, customerID string) (*ProviderBillingSummary, error) {
	return NoopProvider{}.FetchBillingSummary(ctx, customerID)
}

func (p *PaddleProvider) UpdateSubscription(ctx context.Context, req SwitchRequest) error {
	return NoopProvider{}.UpdateSubscription(ctx, req)
}

func (p *PaddleProvider) PreviewSubscriptionUpdate(ctx context.Context, subscriptionID, targetPriceID string) (*ProrationPreview, error) {
	return NoopProvider{}.PreviewSubscriptionUpdate(ctx, subscriptionID, targetPriceID)
}

func (p *PaddleProvider) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return NoopProvider{}.CancelSubscription(ctx, subscriptionID)
}

func (p *PaddleProvider) VerifyWebhook(payload []byte, headers map[string]string, secret string) error {
	sig := headers["Paddle-Signature"]
	sec := secret
	if sec == "" {
		sec = p.webhookSecret
	}
	if sec == "" {
		return errors.New("paddle: webhook secret is not configured")
	}
	return p.verifyPaddleSignature(payload, sig, sec)
}

// HandleWebhook verifies the Paddle-Signature header and dispatches the event.
// Paddle signs webhooks with HMAC-SHA256 using the secret from the dashboard.
func (p *PaddleProvider) HandleWebhook(_ context.Context, event *WebhookEvent) (*WebhookResult, error) {
	// SEC-03: Reject unsigned webhooks — an empty secret means signature
	// verification would be skipped, allowing attackers to forge events.
	if p.webhookSecret == "" {
		return nil, errors.New("paddle: webhook secret is not configured — refusing unsigned event")
	}

	sig := event.Headers["Paddle-Signature"]
	if err := p.verifyPaddleSignature(event.Raw, sig, p.webhookSecret); err != nil {
		return nil, fmt.Errorf("paddle: invalid webhook signature: %w", err)
	}

	// Parse the raw payload.
	var payload map[string]any
	if err := json.Unmarshal(event.Raw, &payload); err != nil {
		return nil, fmt.Errorf("paddle: failed to parse webhook body: %w", err)
	}

	eventType, _ := payload["event_type"].(string)
	result := &WebhookResult{EventType: eventType}

	// Paddle puts the entity in "data".
	data, _ := payload["data"].(map[string]any)
	if data == nil {
		return result, nil
	}

	// Extract org_id from custom_data.
	if customData, ok := data["custom_data"].(map[string]any); ok {
		result.OrgID, _ = customData["org_id"].(string)
	}

	switch eventType {
	case "transaction.completed":
		result.ProviderSubscriptionID, _ = data["subscription_id"].(string)
		result.ProviderCustomerID, _ = data["customer_id"].(string)
		result.Status = "active"

	case "subscription.activated":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.ProviderCustomerID, _ = data["customer_id"].(string)
		result.Status = "active"

	case "subscription.updated":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.ProviderCustomerID, _ = data["customer_id"].(string)
		result.Status, _ = data["status"].(string)
		if scheduledChange, ok := data["scheduled_change"].(map[string]any); ok {
			if action, _ := scheduledChange["action"].(string); action == "cancel" {
				result.CancelAtPeriodEnd = true
			}
		}
		if billingPeriod, ok := data["current_billing_period"].(map[string]any); ok {
			if startStr, ok := billingPeriod["starts_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, startStr); err == nil {
					result.CurrentPeriodStart = &t
				}
			}
			if endStr, ok := billingPeriod["ends_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, endStr); err == nil {
					result.CurrentPeriodEnd = &t
				}
			}
		}

	case "subscription.canceled":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.Status = "canceled"

	case "subscription.past_due":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.Status = "past_due"

	case "subscription.paused":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.Status = "paused"

	case "subscription.resumed":
		result.ProviderSubscriptionID, _ = data["id"].(string)
		result.Status = "active"
	}

	return result, nil
}

// verifyPaddleSignature validates Paddle-Signature header.
// Format: "ts=<unix>;h1=<hex-hmac>"
func (p *PaddleProvider) verifyPaddleSignature(payload []byte, header string, secret string) error {
	parts := strings.Split(header, ";")
	var ts, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "ts=") {
			ts = strings.TrimPrefix(part, "ts=")
		}
		if strings.HasPrefix(part, "h1=") {
			sig = strings.TrimPrefix(part, "h1=")
		}
	}
	if ts == "" || sig == "" {
		return errors.New("malformed Paddle-Signature header")
	}

	signed := ts + ":" + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return errors.New("paddle signature mismatch")
	}
	return nil
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

func (p *PaddleProvider) get(ctx context.Context, path string) (map[string]any, error) {
	req, err := p.client.NewJSONRequest(ctx, http.MethodGet, p.baseURL+path, p.apiKey, nil)
	if err != nil {
		return nil, err
	}
	body, _, err := p.client.DoWithRetry(req, ProviderPaddle)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	return result, json.Unmarshal(body, &result)
}

func (p *PaddleProvider) post(ctx context.Context, path string, body any) (map[string]any, error) {
	req, err := p.client.NewJSONRequest(ctx, http.MethodPost, p.baseURL+path, p.apiKey, body)
	if err != nil {
		return nil, err
	}
	respBody, _, err := p.client.Do(req, ProviderPaddle)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	return result, json.Unmarshal(respBody, &result)
}
