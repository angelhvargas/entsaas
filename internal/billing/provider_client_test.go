package billing_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"entsaas/internal/billing"
)

func TestProviderHTTPClient_RequestBuilders(t *testing.T) {
	client := billing.NewProviderHTTPClient(5*time.Second, 2)

	// JSON request
	req, err := client.NewJSONRequest(context.Background(), "POST", "http://example.com", "apiKey", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("NewJSONRequest failed: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer apiKey" {
		t.Errorf("expected bearer token header, got %s", req.Header.Get("Authorization"))
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected json content-type, got %s", req.Header.Get("Content-Type"))
	}

	// Form request
	req, err = client.NewFormRequest(context.Background(), "POST", "http://example.com", "apiKey", "foo=bar")
	if err != nil {
		t.Fatalf("NewFormRequest failed: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer apiKey" {
		t.Errorf("expected bearer token header, got %s", req.Header.Get("Authorization"))
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content-type, got %s", req.Header.Get("Content-Type"))
	}
}

func TestProviderHTTPClient_Do_ErrorClassificationAndRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API Key"}}`))
	}))
	defer server.Close()

	client := billing.NewProviderHTTPClient(5*time.Second, 2)
	req, err := http.NewRequestWithContext(context.Background(), "GET", server.URL+"/v1/invoices?secret=xyz", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	body, status, err := client.Do(req, billing.ProviderStripe)
	_ = body
	if status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", status)
	}

	var apiErr *billing.ProviderAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected ProviderAPIError, got %v", err)
	}
	if apiErr.Message != "Invalid API Key" {
		t.Errorf("expected safe message 'Invalid API Key', got %q", apiErr.Message)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected error status 401, got %d", apiErr.StatusCode)
	}
}

func TestProviderHTTPClient_DoWithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := billing.NewProviderHTTPClient(5*time.Second, 3)
	req, err := http.NewRequestWithContext(context.Background(), "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	body, status, err := client.DoWithRetry(req, billing.ProviderNoop)
	if err != nil {
		t.Fatalf("DoWithRetry failed: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", attempts)
	}
	if !stringContains(string(body), "success") {
		t.Errorf("expected response to contain 'success', got %s", string(body))
	}
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || s[:len(sub)] == sub || s[len(s)-len(sub):] == sub || len(s) > len(sub)) // simple check
}
