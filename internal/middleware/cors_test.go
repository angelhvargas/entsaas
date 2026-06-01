package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_Default(t *testing.T) {
	// Ensure env var is empty
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request with default allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:5173, got %q", allowOrigin)
	}
}

func TestCORS_CustomOrigins(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://custom-origin.com, https://another-one.net")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Test matching custom origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://custom-origin.com")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://custom-origin.com" {
		t.Errorf("expected allowed origin http://custom-origin.com, got %q", allowOrigin)
	}

	// Test non-matching origin
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "http://malicious.com")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	allowOrigin2 := w2.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin2 != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for untrusted domain, got %q", allowOrigin2)
	}
}
