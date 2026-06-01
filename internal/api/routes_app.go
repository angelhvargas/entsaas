package api

import (
	"entsaas/internal/handlers"
	"entsaas/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAppRoutes registers session-authenticated product UI endpoints.
func RegisterAppRoutes(v1 *gin.RouterGroup, deps RouterDeps) {
	app := v1.Group("")
	app.Use(middleware.SessionAuth())

	// ── Profile ──────────────────────────────────────────────────────────────
	app.GET("/me", handlers.GetCurrentUser(deps.Store))
	app.POST("/auth/logout", handlers.Logout(deps.Store))

	// ── Projects ─────────────────────────────────────────────────────────────
	projectHandler := handlers.NewProjectHandler(deps.Store)
	app.GET("/projects", projectHandler.List)
	app.POST("/projects", projectHandler.Create)
	app.GET("/projects/:id", projectHandler.Get)
	app.PUT("/projects/:id", projectHandler.Update)
	app.DELETE("/projects/:id", projectHandler.Delete)

	// ── Preferences ──────────────────────────────────────────────────────────
	prefsHandler := handlers.NewPrefsHandler(deps.Store)
	app.GET("/preferences", prefsHandler.Get)
	app.PUT("/preferences", prefsHandler.Set)

	// ── Audit Log ────────────────────────────────────────────────────────────
	app.GET("/audit-log", handlers.GetAuditLog(deps.Store))

	// ── AI ────────────────────────────────────────────────────────────────────
	aiHandler := handlers.NewAIHandler(deps.AIConfig, deps.Store)
	if aiHandler != nil {
		app.POST("/ai/chat", aiHandler.Chat)
		app.POST("/ai/complete", aiHandler.Complete)
	}

	// ── Billing ───────────────────────────────────────────────────────────────
	if deps.Billing != nil {
		billingHandler := handlers.NewBillingHandler(deps.Billing, deps.Store)
		app.GET("/billing/plans", billingHandler.GetPlans)
		app.GET("/billing/plan", billingHandler.GetPlan)
		app.GET("/billing/subscription", billingHandler.GetSubscription)
		app.POST("/billing/checkout", billingHandler.CreateCheckout)
		app.POST("/billing/portal", billingHandler.CreatePortal)
		app.GET("/billing/summary", billingHandler.GetBillingSummary)
		app.GET("/billing/invoices", billingHandler.GetInvoices)
		app.GET("/billing/invoices/:id", billingHandler.GetInvoice)
	}
}
