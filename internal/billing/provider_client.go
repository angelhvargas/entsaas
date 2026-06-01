package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ── Provider HTTP Client Utilities ───────────────────────────────────────────
//
// Shared helpers for making authenticated HTTP requests to billing provider
// APIs. Avoids adding heavy SDKs as dependencies — uses net/http with proper
// error handling, structured logging, and response normalization.
//
// Retry/Backoff Policy:
//   - Idempotent GET requests: retry on 429/5xx
//   - Non-idempotent POST requests: NO retry — fail fast
//   - Backoff: exponential (1s, 2s) with context awareness
//   - Timeout: 15s per request via DefaultHTTPClient timeout

const (
	errCategoryConfig  = "config"  // missing API key, invalid config
	errCategoryAuth    = "auth"    // 401/403 from provider
	errCategoryNetwork = "network" // connection refused, timeout, DNS
	errCategoryAPI     = "api"     // 4xx/5xx from provider
	errCategoryParse   = "parse"   // failed to parse provider response
)

// ProviderHTTPClient is an injectable, context-aware HTTP client with built-in
// retry, response body limitation, and structured logging for billing APIs.
type ProviderHTTPClient struct {
	client     *http.Client
	logger     zerolog.Logger
	maxRetries int
}

// NewProviderHTTPClient constructs a new ProviderHTTPClient.
func NewProviderHTTPClient(timeout time.Duration, maxRetries int) *ProviderHTTPClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if maxRetries < 0 {
		maxRetries = 2
	}
	return &ProviderHTTPClient{
		client:     &http.Client{Timeout: timeout},
		logger:     log.With().Str("component", "billing_client").Logger(),
		maxRetries: maxRetries,
	}
}

// ── Error Classification ─────────────────────────────────────────────────────

// classifyError returns a human-readable error category for logging.
func classifyError(statusCode int, err error) string {
	if statusCode == 401 || statusCode == 403 {
		return errCategoryAuth
	}
	if statusCode >= 400 && statusCode < 500 {
		return errCategoryAPI
	}
	if statusCode >= 500 {
		return errCategoryAPI
	}
	if err != nil && statusCode == 0 {
		return errCategoryNetwork
	}
	return errCategoryAPI
}

// ── Request Builders ─────────────────────────────────────────────────────────

// NewJSONRequest builds an authenticated JSON-body request (e.g. Paddle).
func (c *ProviderHTTPClient) NewJSONRequest(ctx context.Context, method, url, apiKey string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("billing: marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("billing: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// NewFormRequest builds an authenticated form-encoded request (e.g. Stripe).
func (c *ProviderHTTPClient) NewFormRequest(ctx context.Context, method, url, apiKey string, form string) (*http.Request, error) {
	var bodyReader io.Reader
	if form != "" {
		bodyReader = strings.NewReader(form)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("billing: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if form != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return req, nil
}

// ── Request Execution ────────────────────────────────────────────────────────

// Do executes a single provider API request and returns the response body, status code, and error.
// Does NOT retry. Use DoWithRetry for idempotent GET requests.
// Logs every call; error responses are logged as warnings.
func (c *ProviderHTTPClient) Do(req *http.Request, provider ProviderName) ([]byte, int, error) {
	start := time.Now()

	resp, err := c.client.Do(req) // #nosec G704 -- safe by definition: URLs constructed programmatically
	duration := time.Since(start)

	if err != nil {
		c.logger.Warn().
			Str("provider", string(provider)).
			Str("method", req.Method).
			Str("path", c.redactURL(req.URL.Path)).
			Dur("duration_ms", duration).
			Str("error_category", errCategoryNetwork).
			Err(err).
			Msg("billing: provider API call failed")
		return nil, 0, providerError(provider, 0, "request failed", err)
	}
	defer resp.Body.Close()

	// 1 MiB cap (SEC-14) to prevent out-of-memory issues from oversized responses
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.logger.Warn().
			Str("provider", string(provider)).
			Str("method", req.Method).
			Int("status", resp.StatusCode).
			Str("error_category", errCategoryParse).
			Msg("billing: failed to read provider response")
		return nil, resp.StatusCode, providerError(provider, resp.StatusCode, "failed to read response", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		safeMsg := c.extractProviderErrorMessage(body, provider)
		category := classifyError(resp.StatusCode, nil)
		c.logger.Warn().
			Str("provider", string(provider)).
			Str("method", req.Method).
			Str("path", c.redactURL(req.URL.Path)).
			Int("status", resp.StatusCode).
			Dur("duration_ms", duration).
			Str("error_category", category).
			Str("error_message", safeMsg).
			Msg("billing: provider API returned error")
		return body, resp.StatusCode, providerError(provider, resp.StatusCode, safeMsg, errors.New(string(body)))
	}

	c.logger.Debug().
		Str("provider", string(provider)).
		Str("method", req.Method).
		Str("path", c.redactURL(req.URL.Path)).
		Int("status", resp.StatusCode).
		Dur("duration_ms", duration).
		Msg("billing: provider API call succeeded")

	return body, resp.StatusCode, nil
}

// DoWithRetry executes an idempotent provider API request with retry functionality.
// Retries on 429 (rate limit) and 5xx (server error) up to maxRetries times
// with exponential backoff (1s, 2s). Non-retryable errors fail immediately.
//
// IMPORTANT: Only use for idempotent GET requests. Never use for POST requests.
func (c *ProviderHTTPClient) DoWithRetry(req *http.Request, provider ProviderName) ([]byte, int, error) {
	var lastBody []byte
	var lastStatus int
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			c.logger.Debug().
				Str("provider", string(provider)).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("billing: retrying provider API call")

			select {
			case <-time.After(backoff):
			case <-req.Context().Done():
				return nil, 0, providerError(provider, 0, "request cancelled during retry backoff", req.Context().Err())
			}
		}

		lastBody, lastStatus, lastErr = c.Do(req, provider)
		if lastErr == nil {
			return lastBody, lastStatus, nil
		}

		// Only retry on 429 (rate limit) or 5xx (server error).
		if lastStatus != http.StatusTooManyRequests && (lastStatus < 500 || lastStatus >= 600) {
			return lastBody, lastStatus, lastErr
		}
	}

	c.logger.Warn().
		Str("provider", string(provider)).
		Int("final_status", lastStatus).
		Int("max_retries", c.maxRetries).
		Msg("billing: provider API call exhausted retries")

	return lastBody, lastStatus, lastErr
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// extractProviderErrorMessage extracts a safe error message from provider error responses.
func (c *ProviderHTTPClient) extractProviderErrorMessage(body []byte, provider ProviderName) string {
	switch provider {
	case ProviderStripe:
		var stripeErr struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &stripeErr) == nil && stripeErr.Error.Message != "" {
			return stripeErr.Error.Message
		}
	case ProviderPaddle:
		var paddleErr struct {
			Error struct {
				Detail string `json:"detail"`
				Code   string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &paddleErr) == nil && paddleErr.Error.Detail != "" {
			return paddleErr.Error.Detail
		}
	}
	return "provider returned an error"
}

// redactURL removes query parameters from a URL path for safe logging.
func (c *ProviderHTTPClient) redactURL(path string) string {
	if before, _, ok := strings.Cut(path, "?"); ok {
		return before
	}
	return path
}
