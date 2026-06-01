package handlers

import (
	"net/http"

	"entsaas/internal/auth"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{ store store.AppStore }

func NewUserHandler(s store.AppStore) *UserHandler { return &UserHandler{store: s} }

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.store.GetUsersByOrg(c.Request.Context(), c.GetString("org_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list users"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !auth.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Valid role required (owner, admin, member, viewer)"}})
		return
	}
	actorRole := c.GetString("role")
	actorID := c.GetString("user_id")
	targetID := c.Param("id")

	// SEC-02: Prevent self-role-downgrade (lockout risk).
	if targetID == actorID {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Cannot change your own role"}})
		return
	}

	if !auth.CanManageRole(actorRole, req.Role) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Cannot assign role equal or higher than your own"}})
		return
	}
	if err := h.store.UpdateUserRole(c.Request.Context(), targetID, c.GetString("org_id"), req.Role); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "User not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

func (h *UserHandler) UpdateStatus(c *gin.Context) {
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "is_active field required"}})
		return
	}
	// SEC-02: Prevent self-deactivation (lockout risk).
	if c.Param("id") == c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Cannot change your own status"}})
		return
	}
	if err := h.store.UpdateUserStatus(c.Request.Context(), c.Param("id"), c.GetString("org_id"), req.IsActive); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "User not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User status updated"})
}
