package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"entsaas/internal/auth"
	"entsaas/internal/bootstrap"
	"entsaas/internal/mail"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

const maxPasswordLength = 128

var slugRegexp = regexp.MustCompile(`[^a-z0-9-]+`)

// prehashPassword applies SHA-256 before bcrypt to avoid the 72-byte
// silent truncation vulnerability. This is the same pattern used by
// Dropbox and Django. The hex output is always 64 bytes, well within
// bcrypt's 72-byte limit.
func prehashPassword(password string) []byte {
	h := sha256.Sum256([]byte(password))
	return []byte(hex.EncodeToString(h[:]))
}

// sanitizeSlug normalises an organisation name into a URL-safe slug.
// Only lowercase alphanumeric characters and hyphens are kept.
// Output is capped at 63 characters (DNS label safe).
func sanitizeSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRegexp.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = s[:63]
	}
	if s == "" {
		s = "org"
	}
	return s
}

// AuthHandler handles all authentication endpoints.
type AuthHandler struct {
	store  store.AppStore
	mailer mail.Sender
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(s store.AppStore, mailer mail.Sender) *AuthHandler {
	if mailer == nil {
		mailer = mail.LogSender{}
	}
	return &AuthHandler{store: s, mailer: mailer}
}

// Login authenticates a user with email/password and returns JWT tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Email and password are required"},
		})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Find all user records for this email.
	users, err := h.store.GetUsersByEmail(c.Request.Context(), req.Email)
	if err != nil || len(users) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"},
		})
		return
	}

	// Use the first active user.
	var user *store.User
	for _, u := range users {
		if u.IsActive {
			user = u
			break
		}
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "ACCOUNT_DISABLED", "message": "Account is disabled"},
		})
		return
	}

	// Verify password.
	cred, err := h.store.GetUserCredential(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"},
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), prehashPassword(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"},
		})
		return
	}

	// Generate tokens.
	accessTTL := time.Duration(bootstrap.GetEnvInt("ENTSAAS_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	accessToken, err := store.GenerateJWT(user.ID, user.OrgID, user.Email, user.Role, accessTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate token"},
		})
		return
	}

	// Generate refresh token.
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate refresh token"},
		})
		return
	}
	refreshToken := hex.EncodeToString(refreshBytes)
	refreshHash := hashToken(refreshToken)
	refreshTTL := 720 * time.Hour // 30 days

	if err := h.store.CreateRefreshToken(c.Request.Context(),
		refreshHash, user.ID, user.OrgID,
		c.GetHeader("User-Agent"), c.ClientIP(), refreshTTL); err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("SEC-07: failed to persist refresh token")
	}

	// Audit log.
	_ = h.store.LogAuditEvent(c.Request.Context(), &user.ID, user.OrgID,
		"user.login", "user", user.ID, map[string]any{"ip": c.ClientIP()})

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTTL.Seconds()),
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
			"org_id": user.OrgID,
		},
	})
}

// Register creates a new organization and user.
func (h *AuthHandler) Register(c *gin.Context) {
	if !bootstrap.GetEnvBool("FF_REGISTRATION_ENABLED", true) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{"code": "REGISTRATION_DISABLED", "message": "Registration is currently disabled"},
		})
		return
	}

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		OrgName  string `json:"org_name" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Valid email, password (≥8 chars), and org_name are required"},
		})
		return
	}

	if len(req.Password) > maxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Password must not exceed 128 characters"},
		})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists.
	existing, _ := h.store.GetUsersByEmail(c.Request.Context(), req.Email)
	if len(existing) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{"code": "EMAIL_EXISTS", "message": "An account with this email already exists"},
		})
		return
	}

	// Create organization.
	slug := sanitizeSlug(req.OrgName)
	org, err := h.store.CreateOrganization(c.Request.Context(), req.OrgName, slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create organization"},
		})
		return
	}

	// Create user with owner role.
	user, err := h.store.CreateUser(c.Request.Context(), req.Email, auth.RoleOwner, org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create user"},
		})
		return
	}

	// Hash and store password.
	hash, err := bcrypt.GenerateFromPassword(prehashPassword(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"},
		})
		return
	}
	if err := h.store.CreateUserCredential(c.Request.Context(), user.ID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to store credentials"},
		})
		return
	}

	// Audit log.
	_ = h.store.LogAuditEvent(c.Request.Context(), &user.ID, org.ID,
		"user.registered", "user", user.ID, map[string]any{"org_name": org.Name})

	// Send welcome email (non-blocking, failure is non-fatal).
	appURL := os.Getenv("ENTSAAS_BASE_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}
	appName := os.Getenv("ENTSAAS_APP_NAME")
	if appName == "" {
		appName = "EntSaaS"
	}
	go func() {
		_ = mail.SendWelcome(h.mailer, user.Email, mail.WelcomeData{
			AppName:  appName,
			AppURL:   appURL,
			UserName: user.Email,
		})
	}()

	// Auto-login: generate access token.
	accessTTL := time.Duration(bootstrap.GetEnvInt("ENTSAAS_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	accessToken, _ := store.GenerateJWT(user.ID, org.ID, user.Email, user.Role, accessTTL)

	c.JSON(http.StatusCreated, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(accessTTL.Seconds()),
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
			"org_id": org.ID,
		},
		"organization": gin.H{
			"id":   org.ID,
			"name": org.Name,
			"slug": org.Slug,
		},
	})
}

// Refresh exchanges a refresh token for a new access token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "refresh_token is required"},
		})
		return
	}

	tokenHash := hashToken(req.RefreshToken)
	rt, err := h.store.GetRefreshToken(c.Request.Context(), tokenHash)
	if err != nil || rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "INVALID_TOKEN", "message": "Invalid or expired refresh token"},
		})
		return
	}

	// Look up user.
	user, err := h.store.GetUserByID(c.Request.Context(), rt.UserID)
	if err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "ACCOUNT_DISABLED", "message": "Account is disabled"},
		})
		return
	}

	// Revoke old refresh token and issue new one (rotation).
	_ = h.store.RevokeRefreshToken(c.Request.Context(), tokenHash)

	accessTTL := time.Duration(bootstrap.GetEnvInt("ENTSAAS_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	accessToken, _ := store.GenerateJWT(user.ID, user.OrgID, user.Email, user.Role, accessTTL)

	// Issue new refresh token.
	refreshBytes := make([]byte, 32)
	_, _ = rand.Read(refreshBytes)
	newRefreshToken := hex.EncodeToString(refreshBytes)
	newRefreshHash := hashToken(newRefreshToken)
	if err := h.store.CreateRefreshToken(c.Request.Context(),
		newRefreshHash, user.ID, user.OrgID,
		c.GetHeader("User-Agent"), c.ClientIP(), 720*time.Hour); err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("SEC-07: failed to persist rotated refresh token")
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTTL.Seconds()),
	})
}

// ForgotPassword initiates a password reset flow.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Valid email is required"},
		})
		return
	}

	// Always return success to prevent email enumeration.
	users, err := h.store.GetUsersByEmail(c.Request.Context(), strings.ToLower(req.Email))
	if err == nil && len(users) > 0 {
		tokenBytes := make([]byte, 32)
		_, _ = rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)
		tokenHash := hashToken(token)
		_ = h.store.CreateResetToken(c.Request.Context(), tokenHash, users[0].ID)

		// Send password reset email (non-blocking, failure is non-fatal).
		baseURL := os.Getenv("ENTSAAS_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:5173"
		}
		appName := os.Getenv("ENTSAAS_APP_NAME")
		if appName == "" {
			appName = "EntSaaS"
		}
		resetURL := baseURL + "/reset-password?token=" + token
		go func() {
			_ = mail.SendPasswordReset(h.mailer, req.Email, mail.PasswordResetData{
				AppName:   appName,
				AppURL:    baseURL,
				ResetURL:  resetURL,
				ExpiresIn: "1 hour",
			})
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account with that email exists, a password reset link has been sent",
	})
}

// ResetPassword completes the password reset flow.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Token and password (≥8 chars) are required"},
		})
		return
	}

	if len(req.Password) > maxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Password must not exceed 128 characters"},
		})
		return
	}

	tokenHash := hashToken(req.Token)
	rt, err := h.store.ConsumeResetToken(c.Request.Context(), tokenHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "INVALID_TOKEN", "message": "Invalid or expired reset token"},
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword(prehashPassword(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"},
		})
		return
	}

	if err := h.store.UpdateUserCredential(c.Request.Context(), rt.UserID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update password"},
		})
		return
	}

	// Revoke all refresh tokens for security.
	_ = h.store.RevokeAllUserRefreshTokens(c.Request.Context(), rt.UserID)

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}

// VerifyEmail confirms a user's email address.
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Token is required"},
		})
		return
	}

	tokenHash := hashToken(req.Token)
	vt, err := h.store.GetVerificationTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || vt.UsedAt != nil || time.Now().After(vt.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "INVALID_TOKEN", "message": "Invalid or expired verification token"},
		})
		return
	}

	_ = h.store.MarkVerificationTokenUsed(c.Request.Context(), tokenHash)
	_ = h.store.MarkUserEmailVerified(c.Request.Context(), vt.UserID)

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// GetCurrentUser returns the authenticated user's profile.
func GetCurrentUser(s store.AppStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		user, err := s.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{"code": "NOT_FOUND", "message": "User not found"},
			})
			return
		}

		org, _ := s.GetOrganizationByID(c.Request.Context(), user.OrgID)

		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":                user.ID,
				"email":             user.Email,
				"role":              user.Role,
				"org_id":            user.OrgID,
				"is_active":         user.IsActive,
				"email_verified_at": user.EmailVerifiedAt,
				"created_at":        user.CreatedAt,
			},
			"organization": org,
		})
	}
}

// Logout revokes the user's current refresh token.
func Logout(s store.AppStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		_ = s.RevokeAllUserRefreshTokens(c.Request.Context(), userID)
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

// GetConfig returns deployment feature flags for the frontend.
func GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"registration_enabled":       bootstrap.GetEnvBool("FF_REGISTRATION_ENABLED", true),
		"email_verification_enabled": bootstrap.GetEnvBool("FF_EMAIL_VERIFICATION_ENABLED", false),
		"ai_enabled":                 bootstrap.GetEnvBool("AI_ENABLED", false),
	})
}

// GetAuditLog returns the org's audit log with cursor-based pagination.
func GetAuditLog(s store.AppStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID := c.GetString("org_id")
		cursor := c.Query("cursor")
		limit := bootstrap.GetEnvInt("AUDIT_LOG_PAGE_SIZE", 50)

		entries, hasMore, err := s.GetAuditLogs(c.Request.Context(), orgID, limit, cursor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to fetch audit log"},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"entries":  entries,
			"has_more": hasMore,
		})
	}
}

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
