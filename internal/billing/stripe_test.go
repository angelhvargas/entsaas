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

func generateStripeSignature(payload []byte, secret string, timestamp int64) string {
	tsStr := strconv.FormatInt(timestamp, 10)
	signed := tsStr + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	return "t=" + tsStr + ",v1=" + sig
}

func TestNewStripeProvider(t *testing.T) {
	// Without env var
	os.Setenv("STRIPE_SECRET_KEY", "")
	p, err := billing.NewStripeProvider()
	if err == nil || p != nil {
		t.Error("expected error when STRIPE_SECRET_KEY is empty")
	}

	// With env var
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	p, err = billing.NewStripeProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != billing.ProviderStripe {
		t.Errorf("expected provider name stripe, got %s", p.Name())
	}
}

func TestStripeProvider_Stubs(t *testing.T) {
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	p, err := billing.NewStripeProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	plans, err := p.ListPlans(ctx)
	if err != nil || len(plans) == 0 {
		t.Errorf("ListPlans stub failed: %v", err)
	}

	sub, err := p.GetSubscription(ctx, "org_1")
	if err != nil || sub == nil {
		t.Errorf("GetSubscription stub failed: %v", err)
	}

	checkout, err := p.CreateCheckoutSession(ctx, "org_1", "pro", "http://success", "http://cancel")
	if err != nil || checkout == nil {
		t.Errorf("CreateCheckoutSession stub failed: %v", err)
	}

	portal, err := p.CreatePortalSession(ctx, "cus_123")
	if err != nil || portal == nil {
		t.Errorf("CreatePortalSession stub failed: %v", err)
	}

	summary, err := p.FetchBillingSummary(ctx, "cus_123")
	if err != nil || summary == nil {
		t.Errorf("FetchBillingSummary stub failed: %v", err)
	}

	err = p.UpdateSubscription(ctx, billing.SwitchRequest{})
	if err != nil {
		t.Errorf("UpdateSubscription stub failed: %v", err)
	}

	proration, err := p.PreviewSubscriptionUpdate(ctx, "sub_1", "price_2")
	if err != nil || proration == nil {
		t.Errorf("PreviewSubscriptionUpdate stub failed: %v", err)
	}

	err = p.CancelSubscription(ctx, "sub_1")
	if err != nil {
		t.Errorf("CancelSubscription stub failed: %v", err)
	}
}

func TestStripeProvider_VerifyWebhook(t *testing.T) {
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	p, err := billing.NewStripeProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := []byte(`{"type": "ping"}`)
	secret := "whsec_123"
	ts := time.Now().Unix()

	// Valid signature
	sig := generateStripeSignature(payload, secret, ts)
	headers := map[string]string{"Stripe-Signature": sig}
	err = p.VerifyWebhook(payload, headers, secret)
	if err != nil {
		t.Errorf("VerifyWebhook failed on valid signature: %v", err)
	}

	// Invalid signature
	headers = map[string]string{"Stripe-Signature": "t=123,v1=bad"}
	err = p.VerifyWebhook(payload, headers, secret)
	if err == nil {
		t.Error("expected error on invalid signature")
	}

	// Malformed signature
	headers = map[string]string{"Stripe-Signature": "bad"}
	err = p.VerifyWebhook(payload, headers, secret)
	if err == nil {
		t.Error("expected error on malformed signature")
	}

	// Out of tolerance signature (timestamp too old)
	oldTs := time.Now().Unix() - 600 // 10 minutes ago
	sig = generateStripeSignature(payload, secret, oldTs)
	headers = map[string]string{"Stripe-Signature": sig}
	err = p.VerifyWebhook(payload, headers, secret)
	if err == nil {
		t.Error("expected error on expired timestamp signature")
	}
}

func TestStripeProvider_HandleWebhook(t *testing.T) {
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	p, err := billing.NewStripeProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	secret := "whsec_123"

	// 1. checkout.session.completed
	payloadCheckout := []byte(`{
		"type": "checkout.session.completed",
		"data": {
			"object": {
				"customer": "cus_stripe_1",
				"subscription": "sub_stripe_1",
				"client_reference_id": "org_checkout_1",
				"metadata": {
					"org_id": "org_checkout_1"
				}
			}
		}
	}`)
	sig := generateStripeSignature(payloadCheckout, secret, time.Now().Unix())
	event := &billing.WebhookEvent{
		Raw:     payloadCheckout,
		Headers: map[string]string{"Stripe-Signature": sig},
	}
	res, err := p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "checkout.session.completed" || res.ProviderCustomerID != "cus_stripe_1" || res.OrgID != "org_checkout_1" {
		t.Errorf("incorrect checkout webhook result: %+v", res)
	}

	// 2. customer.subscription.updated
	payloadUpdate := []byte(`{
		"type": "customer.subscription.updated",
		"data": {
			"object": {
				"id": "sub_stripe_1",
				"customer": "cus_stripe_1",
				"status": "active",
				"cancel_at_period_end": true,
				"current_period_start": 1700000000,
				"current_period_end": 1700086400,
				"metadata": {
					"org_id": "org_checkout_1"
				}
			}
		}
	}`)
	sig = generateStripeSignature(payloadUpdate, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadUpdate,
		Headers: map[string]string{"Stripe-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "customer.subscription.updated" || res.Status != "active" || !res.CancelAtPeriodEnd || res.CurrentPeriodStart == nil {
		t.Errorf("incorrect update webhook result: %+v", res)
	}

	// 3. customer.subscription.deleted
	payloadDelete := []byte(`{
		"type": "customer.subscription.deleted",
		"data": {
			"object": {
				"id": "sub_stripe_1",
				"metadata": {
					"org_id": "org_checkout_1"
				}
			}
		}
	}`)
	sig = generateStripeSignature(payloadDelete, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadDelete,
		Headers: map[string]string{"Stripe-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "customer.subscription.deleted" || res.Status != "canceled" || res.OrgID != "org_checkout_1" {
		t.Errorf("incorrect delete webhook result: %+v", res)
	}

	// 4. invoice.payment_failed
	payloadFailed := []byte(`{
		"type": "invoice.payment_failed",
		"data": {
			"object": {
				"subscription": "sub_stripe_1",
				"customer": "cus_stripe_1"
			}
		}
	}`)
	sig = generateStripeSignature(payloadFailed, secret, time.Now().Unix())
	event = &billing.WebhookEvent{
		Raw:     payloadFailed,
		Headers: map[string]string{"Stripe-Signature": sig},
	}
	res, err = p.HandleWebhook(ctx, event)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if res.EventType != "invoice.payment_failed" || res.Status != "past_due" || res.ProviderSubscriptionID != "sub_stripe_1" {
		t.Errorf("incorrect failed payment webhook result: %+v", res)
	}
}
