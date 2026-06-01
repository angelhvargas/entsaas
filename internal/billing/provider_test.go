package billing_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"entsaas/internal/billing"
)

func TestProviderAPIError(t *testing.T) {
	underlying := errors.New("underlying failure")

	// Error with status code
	err := &billing.ProviderAPIError{
		Provider:   billing.ProviderStripe,
		StatusCode: 400,
		Message:    "Invalid request parameters",
		Underlying: underlying,
	}

	expectedStr := "billing: stripe API error (HTTP 400): Invalid request parameters"
	if err.Error() != expectedStr {
		t.Errorf("expected error string %q, got %q", expectedStr, err.Error())
	}
	if !errors.Is(err.Unwrap(), underlying) {
		t.Errorf("expected unwrapped error to be underlying, got %v", err.Unwrap())
	}

	// Error without status code
	errNoStatus := &billing.ProviderAPIError{
		Provider:   billing.ProviderPaddle,
		StatusCode: 0,
		Message:    "Network error",
		Underlying: underlying,
	}

	expectedStrNoStatus := "billing: paddle API error: Network error"
	if errNoStatus.Error() != expectedStrNoStatus {
		t.Errorf("expected error string %q, got %q", expectedStrNoStatus, errNoStatus.Error())
	}
}

func TestAdvisoryStatus(t *testing.T) {
	valid := []billing.AdvisoryStatus{
		billing.AdvisoryPaymentOK,
		billing.AdvisoryPaymentWarning,
		billing.AdvisoryPaymentPastDue,
		billing.AdvisoryPaymentMethodExpiring,
		billing.AdvisoryCancellationPending,
		billing.AdvisoryTrialExpiring,
		billing.AdvisoryCheckoutPending,
	}

	for _, s := range valid {
		if !billing.IsValidAdvisoryStatus(s) {
			t.Errorf("expected %s to be valid", s)
		}
	}

	if billing.IsValidAdvisoryStatus("invalid_status") {
		t.Error("expected invalid_status to be invalid")
	}
}

func TestNoopProvider(t *testing.T) {
	p := billing.NoopProvider{}

	if p.Name() != billing.ProviderNoop {
		t.Errorf("expected name 'noop', got %s", p.Name())
	}

	ctx := context.Background()

	plans, err := p.ListPlans(ctx)
	if err != nil || len(plans) == 0 {
		t.Errorf("ListPlans failed: %v", err)
	}

	sub, err := p.GetSubscription(ctx, "org_1")
	if err != nil || sub == nil || sub.PlanID != "free" {
		t.Errorf("GetSubscription failed: %v", err)
	}

	checkout, err := p.CreateCheckoutSession(ctx, "org_1", "pro", "http://success", "http://cancel")
	if err != nil || checkout == nil {
		t.Errorf("CreateCheckoutSession failed: %v", err)
	}

	portal, err := p.CreatePortalSession(ctx, "cus_1")
	if err != nil || portal == nil || portal.PortalURL != "#" {
		t.Errorf("CreatePortalSession failed: %v", err)
	}

	summary, err := p.FetchBillingSummary(ctx, "cus_1")
	if err != nil || summary == nil || len(summary.Invoices) == 0 {
		t.Errorf("FetchBillingSummary failed: %v", err)
	}

	err = p.UpdateSubscription(ctx, billing.SwitchRequest{})
	if err != nil {
		t.Errorf("UpdateSubscription failed: %v", err)
	}

	proration, err := p.PreviewSubscriptionUpdate(ctx, "sub_1", "price_2")
	if err != nil || proration == nil {
		t.Errorf("PreviewSubscriptionUpdate failed: %v", err)
	}

	err = p.CancelSubscription(ctx, "sub_1")
	if err != nil {
		t.Errorf("CancelSubscription failed: %v", err)
	}

	err = p.VerifyWebhook(nil, nil, "")
	if err != nil {
		t.Errorf("VerifyWebhook failed: %v", err)
	}

	res, err := p.HandleWebhook(ctx, &billing.WebhookEvent{Type: "ping"})
	if err != nil || res == nil || res.EventType != "ping" {
		t.Errorf("HandleWebhook failed: %v", err)
	}
}

func TestNewFactory(t *testing.T) {
	// 1. Stripe priority
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("PADDLE_API_KEY", "pk_test_123")
	p := billing.New()
	if p.Name() != billing.ProviderStripe {
		t.Errorf("expected Stripe provider under Stripe Priority, got %s", p.Name())
	}

	// 2. Paddle priority (no Stripe)
	os.Setenv("STRIPE_SECRET_KEY", "")
	p = billing.New()
	if p.Name() != billing.ProviderPaddle {
		t.Errorf("expected Paddle provider under Paddle Priority, got %s", p.Name())
	}

	// 3. Noop priority (no keys)
	os.Setenv("PADDLE_API_KEY", "")
	p = billing.New()
	if p.Name() != billing.ProviderNoop {
		t.Errorf("expected Noop provider when no keys set, got %s", p.Name())
	}
}
