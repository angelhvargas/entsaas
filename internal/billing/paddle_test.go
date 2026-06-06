package billing_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"testing"
	"time"

	"entsaas/internal/billing"
)

func generatePaddleSignature(payload []byte, secret string, timestamp int64) string {
	tsStr := strconv.FormatInt(timestamp, 10)
	signed := tsStr + ":" + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	return "ts=" + tsStr + ";h1=" + sig
}

func TestNewPaddleProvider(t *testing.T) {
	// Without env var
	os.Setenv("PADDLE_API_KEY", "")
	p, err := billing.NewPaddleProvider()
	if err == nil || p != nil {
		t.Error("expected error when PADDLE_API_KEY is empty")
	}

	// With env var
	os.Setenv("PADDLE_API_KEY", "pk_test_123")
	os.Setenv("PADDLE_WEBHOOK_SECRET", "pwhsec_123")
	p, err = billing.NewPaddleProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != billing.ProviderPaddle {
		t.Errorf("expected provider name paddle, got %s", p.Name())
	}
}


func TestPaddleProvider_VerifyWebhook(t *testing.T) {
	os.Setenv("PADDLE_API_KEY", "pk_test_123")
	os.Setenv("PADDLE_WEBHOOK_SECRET", "pwhsec_123")
	p, err := billing.NewPaddleProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := []byte(`{"event_type": "ping"}`)
	secret := "pwhsec_123"
	ts := time.Now().Unix()

	// Valid signature
	sig := generatePaddleSignature(payload, secret, ts)
	headers := map[string]string{"Paddle-Signature": sig}
	err = p.VerifyWebhook(payload, headers, secret)
	if err != nil {
		t.Errorf("VerifyWebhook failed on valid signature: %v", err)
	}

	// Invalid signature
	headers = map[string]string{"Paddle-Signature": "ts=123;h1=bad"}
	err = p.VerifyWebhook(payload, headers, secret)
	if err == nil {
		t.Error("expected error on invalid signature")
	}

	// Malformed signature
	headers = map[string]string{"Paddle-Signature": "bad"}
	err = p.VerifyWebhook(payload, headers, secret)
	if err == nil {
		t.Error("expected error on malformed signature")
	}
}

func TestPaddleProvider_HandleWebhook(t *testing.T) {
	os.Setenv("PADDLE_API_KEY", "pk_test_123")
	os.Setenv("PADDLE_WEBHOOK_SECRET", "pwhsec_123")
	p, err := billing.NewPaddleProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	secret := "pwhsec_123"

	// 1. transaction.completed
	payloadTx := []byte(`{
		"event_type": "transaction.completed",
		"data": {
			"id": "txn_123",
			"subscription_id": "sub_paddle_1",
			"customer_id": "cus_paddle_1",
			"custom_data": {
				"org_id": "org_paddle_1"
			}
		}
	}`)
	sig := generatePaddleSignature(payloadTx, secret, time.Now().Unix())
	event := &billing.WebhookEvent{
		Raw:     payloadTx,
		Headers: map[string]string{"Paddle-Signature": sig},
	}
	res, err := p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "transaction.completed" || res.OrgID != "org_paddle_1" || res.ProviderSubscriptionID != "sub_paddle_1" || res.Status != "active" {
		t.Errorf("incorrect transaction webhook result: %+v", res)
	}

	// 2. subscription.activated
	payloadActivated := []byte(`{
		"event_type": "subscription.activated",
		"data": {
			"id": "sub_paddle_1",
			"customer_id": "cus_paddle_1",
			"custom_data": {
				"org_id": "org_paddle_1"
			}
		}
	}`)
	sig = generatePaddleSignature(payloadActivated, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadActivated,
		Headers: map[string]string{"Paddle-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "subscription.activated" || res.ProviderSubscriptionID != "sub_paddle_1" || res.Status != "active" {
		t.Errorf("incorrect activated webhook result: %+v", res)
	}

	// 3. subscription.updated
	payloadUpdated := []byte(`{
		"event_type": "subscription.updated",
		"data": {
			"id": "sub_paddle_1",
			"customer_id": "cus_paddle_1",
			"status": "paused",
			"custom_data": {
				"org_id": "org_paddle_1"
			},
			"scheduled_change": {
				"action": "cancel"
			},
			"current_billing_period": {
				"starts_at": "2026-06-01T12:00:00Z",
				"ends_at": "2026-07-01T12:00:00Z"
			}
		}
	}`)
	sig = generatePaddleSignature(payloadUpdated, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadUpdated,
		Headers: map[string]string{"Paddle-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "subscription.updated" || res.Status != "paused" || !res.CancelAtPeriodEnd || res.CurrentPeriodStart == nil {
		t.Errorf("incorrect updated webhook result: %+v", res)
	}

	// 4. subscription.canceled
	payloadCanceled := []byte(`{
		"event_type": "subscription.canceled",
		"data": {
			"id": "sub_paddle_1"
		}
	}`)
	sig = generatePaddleSignature(payloadCanceled, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadCanceled,
		Headers: map[string]string{"Paddle-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "subscription.canceled" || res.Status != "canceled" {
		t.Errorf("incorrect canceled webhook result: %+v", res)
	}
}
