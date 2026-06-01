package api

import (
	"entsaas/internal/handlers"
	"entsaas/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers org-admin elevated endpoints.
func RegisterAdminRoutes(v1 *gin.RouterGroup, deps RouterDeps) {
	admin := v1.Group("/admin")
	admin.Use(middleware.SessionAuth())
	admin.Use(middleware.RequireRole("owner", "admin"))

	// ── User Management ──────────────────────────────────────────────────────
	userHandler := handlers.NewUserHandler(deps.Store)
	admin.GET("/users", userHandler.List)
	admin.PUT("/users/:id/role", userHandler.UpdateRole)
	admin.PUT("/users/:id/status", userHandler.UpdateStatus)

	// ── Invite Management ────────────────────────────────────────────────────────
	inviteHandler := handlers.NewInviteHandler(deps.Store, deps.Mailer)
	admin.GET("/invites", inviteHandler.List)
	admin.POST("/invites", inviteHandler.Create)
	admin.DELETE("/invites/:id", inviteHandler.Delete)
}
