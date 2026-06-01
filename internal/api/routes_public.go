package api

import (
	"time"

	"entsaas/internal/handlers"
	"entsaas/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers unauthenticated endpoints.
// These are accessible without a valid session token.
func RegisterPublicRoutes(r *gin.Engine, v1 *gin.RouterGroup, deps RouterDeps) {
	// ── Health ───────────────────────────────────────────────────────────────
	r.GET("/healthz", handlers.Healthz)
	r.GET("/readyz", handlers.Readyz(deps.Store))

	// ── Auth ─────────────────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	auth.Use(middleware.RateLimit(20, 1*time.Minute)) // SEC-13: 20 req/min per IP
	{
		authHandler := handlers.NewAuthHandler(deps.Store, deps.Mailer)
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.POST("/verify-email", authHandler.VerifyEmail)
	}

	// ── Config (deployment feature flags) ────────────────────────────────────
	v1.GET("/config", handlers.GetConfig)

	// ── Invite Accept (unauthenticated) ──────────────────────────────────
	// Peek: let the accept page show invite info before the user sets a password.
	// Accept: consumes the token, creates the user if new, returns an access token.
	inviteHandler := handlers.NewInviteHandler(deps.Store, deps.Mailer)
	v1.GET("/invites/peek", inviteHandler.PeekInvite)
	v1.POST("/invites/accept", inviteHandler.AcceptInvite)

	// ── Billing Webhook (unauthenticated) ──────────────────────────────────
	if deps.Billing != nil {
		billingHandler := handlers.NewBillingHandler(deps.Billing, deps.Store)
		v1.POST("/billing/webhook", billingHandler.Webhook)
	}
}
