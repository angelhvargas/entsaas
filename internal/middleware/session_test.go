package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

func initJWTSecret() {
	os.Setenv("JWT_SECRET", "11111111111111111111111111111111")
}

func TestSessionAuth_MissingHeader(t *testing.T) {
	initJWTSecret()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach here")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestSessionAuth_MalformedHeader(t *testing.T) {
	initJWTSecret()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach here")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic credentials")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestSessionAuth_SuccessAndRoleVerification(t *testing.T) {
	initJWTSecret()
	token, err := store.GenerateJWT("user-1", "org-1", "test@entsaas.dev", "admin", 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate test JWT: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SessionAuth())

	// Route requiring admin role
	r.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		oid, _ := c.Get("org_id")
		email, _ := c.Get("email")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{
			"user_id": uid,
			"org_id":  oid,
			"email":   email,
			"role":    role,
		})
	})

	// Route requiring member role (should forbid admin in this configuration if role doesn't match)
	r.GET("/member-only", RequireRole("member"), func(c *gin.Context) {
		c.String(http.StatusOK, "member panel")
	})

	// 1. Verify successful auth & claims injection on /admin
	req1 := httptest.NewRequest("GET", "/admin", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 OK on admin route, got %d", w1.Code)
	}

	// 2. Verify RequireRole blocks insufficient privileges (admin tries to access member-only)
	req2 := httptest.NewRequest("GET", "/member-only", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", w2.Code)
	}
}

func TestRequireRole_NoAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Skip SessionAuth to simulate a route missing session credentials but having RequireRole
	r.GET("/unauthenticated-role-check", RequireRole("admin"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/unauthenticated-role-check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}
