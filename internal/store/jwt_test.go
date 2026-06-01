package store

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestGenerateAndVerifyJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "00000000000000000000000000000000")
	defer os.Unsetenv("JWT_SECRET")

	// Reset the sync.Once for test isolation.
	jwtSecret = nil
	jwtSecretMu = sync.Once{}

	token, err := GenerateJWT("user-123", "org-456", "test@example.com", "admin", 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	claims, err := VerifyJWT(token)
	if err != nil {
		t.Fatalf("VerifyJWT failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.OrgID != "org-456" {
		t.Errorf("OrgID = %q, want %q", claims.OrgID, "org-456")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "test@example.com")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
}

