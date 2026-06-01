package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_DevMode(t *testing.T) {
	os.Setenv("ENTSAAS_DEV", "1")
	defer os.Unsetenv("ENTSAAS_DEV")
	os.Unsetenv("CSP_REPORT_URI")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Validate status
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	headers := w.Header()

	// Assert essential security headers
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %q", headers.Get("X-Frame-Options"))
	}
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %q", headers.Get("X-Content-Type-Options"))
	}
	// SEC-12: X-XSS-Protection is intentionally absent (deprecated, can introduce XSS in legacy IE)
	if xss := headers.Get("X-XSS-Protection"); xss != "" {
		t.Errorf("expected X-XSS-Protection to be absent (SEC-12), got %q", xss)
	}
	if headers.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %q", headers.Get("Referrer-Policy"))
	}
	if headers.Get("Permissions-Policy") != "camera=(), microphone=(), geolocation=()" {
		t.Errorf("expected Permissions-Policy, got %q", headers.Get("Permissions-Policy"))
	}

	// Assert HSTS is NOT present in dev mode
	if hsts := headers.Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("expected HSTS to be absent in dev mode, got %q", hsts)
	}

	// Assert default CSP without report-uri
	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected CSP header to be present")
	}
	if strings.Contains(csp, "report-uri") {
		t.Errorf("expected CSP not to contain report-uri, got %q", csp)
	}
}

func TestSecurityHeaders_ProdAndCSPReportMode(t *testing.T) {
	os.Unsetenv("ENTSAAS_DEV")
	os.Setenv("CSP_REPORT_URI", "https://endpoint.report-uri.com")
	defer os.Unsetenv("CSP_REPORT_URI")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	headers := w.Header()

	// HSTS should be present in prod mode (ENTSAAS_DEV is not 1)
	hsts := headers.Get("Strict-Transport-Security")
	if hsts != "max-age=63072000; includeSubDomains; preload" {
		t.Errorf("expected HSTS in prod, got %q", hsts)
	}

	// CSP should contain report-uri
	csp := headers.Get("Content-Security-Policy")
	if !strings.Contains(csp, "report-uri https://endpoint.report-uri.com") {
		t.Errorf("expected CSP to contain report-uri endpoint, got %q", csp)
	}
}
