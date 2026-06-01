package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"entsaas/internal/auth"
	"entsaas/internal/billing"
	"entsaas/internal/bootstrap"
	"entsaas/internal/mail"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type InviteHandler struct {
	store  store.AppStore
	mailer mail.Sender
}

func NewInviteHandler(s store.AppStore, mailer mail.Sender) *InviteHandler {
	if mailer == nil {
		mailer = mail.LogSender{}
	}
	return &InviteHandler{store: s, mailer: mailer}
}

// List returns all pending invites for the caller's org.
func (h *InviteHandler) List(c *gin.Context) {
	invites, err := h.store.GetInvitesByOrg(c.Request.Context(), c.GetString("org_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list invites"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invites": invites})
}

// Create sends a new invite email and stores the invite.
func (h *InviteHandler) Create(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Email and role required"}})
		return
	}

	// Validate role.
	req.Role = strings.ToLower(req.Role)
	if req.Role != auth.RoleMember && req.Role != auth.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Role must be 'member' or 'admin'"}})
		return
	}

	// Enforce member quota from billing plan.
	enforcer := billing.NewEnforcer(h.store)
	if err := enforcer.CheckMemberLimit(c.Request.Context(), c.GetString("org_id")); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{"code": "QUOTA_EXCEEDED", "message": err.Error()},
		})
		return
	}

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashToken(rawToken)

	invite, err := h.store.CreateInvite(c.Request.Context(),
		c.GetString("org_id"), req.Email, req.Role, tokenHash,
		time.Now().Add(7*24*time.Hour), c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create invite"}})
		return
	}

	// Resolve org name for the email.
	org, _ := h.store.GetOrganizationByID(c.Request.Context(), invite.OrgID)
	orgName := invite.OrgID
	if org != nil {
		orgName = org.Name
	}
	inviterEmail := c.GetString("user_email")
	if inviterEmail == "" {
		inviterEmail = "a teammate"
	}

	baseURL := os.Getenv("ENTSAAS_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	appName := os.Getenv("ENTSAAS_APP_NAME")
	if appName == "" {
		appName = "EntSaaS"
	}

	go func() {
		_ = mail.SendInvite(h.mailer, invite.Email, mail.InviteData{
			AppName:      appName,
			AppURL:       baseURL,
			InviteURL:    baseURL + "/accept-invite?token=" + rawToken,
			OrgName:      orgName,
			InviterEmail: inviterEmail,
			ExpiresIn:    "7 days",
		})
	}()

	_ = h.store.LogAuditEvent(c.Request.Context(), func() *string { s := c.GetString("user_id"); return &s }(),
		invite.OrgID, "invite.created", "invite", invite.ID,
		map[string]any{"email": invite.Email, "role": invite.Role})

	c.JSON(http.StatusCreated, gin.H{"invite": invite})
}

// Delete revokes a pending invite.
func (h *InviteHandler) Delete(c *gin.Context) {
	if err := h.store.DeleteInvite(c.Request.Context(), c.Param("id"), c.GetString("org_id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Invite not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Invite revoked"})
}

// PeekInvite returns non-sensitive invite info for the accept page (no auth required).
func (h *InviteHandler) PeekInvite(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "token query param required"}})
		return
	}
	tokenHash := hashToken(token)
	inv, err := h.store.GetInviteByTokenHash(c.Request.Context(), tokenHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "INVITE_NOT_FOUND", "message": "Invite not found or expired"}})
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": gin.H{"code": "INVITE_EXPIRED", "message": "Invite has expired"}})
		return
	}

	org, _ := h.store.GetOrganizationByID(c.Request.Context(), inv.OrgID)
	orgName := inv.OrgID
	if org != nil {
		orgName = org.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"invite": gin.H{
			"email":      inv.Email,
			"role":       inv.Role,
			"org_name":   orgName,
			"expires_at": inv.ExpiresAt,
		},
	})
}

// AcceptInvite consumes an invite token and either:
//   - Creates a new user + org membership and auto-logs them in, OR
//   - Adds the role to an existing user in the invited org.
func (h *InviteHandler) AcceptInvite(c *gin.Context) {
	var req struct {
		Token    string `json:"token"    binding:"required"`
		Password string `json:"password"` // required only for new accounts
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "token is required"}})
		return
	}

	tokenHash := hashToken(req.Token)
	inv, err := h.store.ConsumeInvite(c.Request.Context(), tokenHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVITE_INVALID", "message": "Invite is invalid or has expired"}})
		return
	}

	ctx := c.Request.Context()

	// Check if user already exists in this org.
	existingUser, _ := h.store.GetUserByEmail(ctx, inv.OrgID, inv.Email)
	if existingUser != nil {
		// User already exists — just update their role and return a token.
		_ = h.store.UpdateUserRole(ctx, existingUser.ID, inv.OrgID, inv.Role)
		_ = h.store.LogAuditEvent(ctx, &existingUser.ID, inv.OrgID,
			"invite.accepted", "user", existingUser.ID, map[string]any{"role": inv.Role})

		accessTTL := time.Duration(bootstrap.GetEnvInt("ENTSAAS_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
		accessToken, _ := store.GenerateJWT(existingUser.ID, inv.OrgID, existingUser.Email, inv.Role, accessTTL)
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   int(accessTTL.Seconds()),
			"user": gin.H{
				"id": existingUser.ID, "email": existingUser.Email,
				"role": inv.Role, "org_id": inv.OrgID,
			},
		})
		return
	}

	// New user — password is required.
	if len(req.Password) < 8 || len(req.Password) > maxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "password must be 8–128 characters"}})
		return
	}

	// Create the user.
	user, err := h.store.CreateUser(ctx, inv.Email, inv.Role, inv.OrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create user"}})
		return
	}

	// Hash and store password.
	hash, err := bcrypt.GenerateFromPassword(prehashPassword(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"}})
		return
	}
	if err := h.store.CreateUserCredential(ctx, user.ID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to store credentials"}})
		return
	}

	_ = h.store.LogAuditEvent(ctx, &user.ID, inv.OrgID,
		"invite.accepted", "user", user.ID, map[string]any{"role": inv.Role, "new_user": true})

	accessTTL := time.Duration(bootstrap.GetEnvInt("ENTSAAS_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	accessToken, _ := store.GenerateJWT(user.ID, inv.OrgID, user.Email, user.Role, accessTTL)

	c.JSON(http.StatusCreated, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(accessTTL.Seconds()),
		"user": gin.H{
			"id": user.ID, "email": user.Email,
			"role": user.Role, "org_id": inv.OrgID,
		},
	})
}
