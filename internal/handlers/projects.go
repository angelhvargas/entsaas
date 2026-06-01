package handlers

import (
	"net/http"

	"entsaas/internal/billing"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

// ProjectHandler manages multi-tenant project/workspace CRUD.
type ProjectHandler struct {
	store store.AppStore
}

func NewProjectHandler(s store.AppStore) *ProjectHandler {
	return &ProjectHandler{store: s}
}

func (h *ProjectHandler) List(c *gin.Context) {
	orgID := c.GetString("org_id")
	projects, err := h.store.ListProjectsByOrg(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list projects"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Project name required (≥2 chars)"}})
		return
	}
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")

	// Quota check
	enforcer := billing.NewEnforcer(h.store)
	if err := enforcer.CheckProjectLimit(c.Request.Context(), orgID); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{"code": "QUOTA_EXCEEDED", "message": err.Error()},
		})
		return
	}

	project, err := h.store.CreateProject(c.Request.Context(), req.Name, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create project"}})
		return
	}
	_ = h.store.LogAuditEvent(c.Request.Context(), &userID, orgID, "project.created", "project", project.ID, map[string]any{"name": project.Name})
	c.JSON(http.StatusCreated, gin.H{"project": project})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	projectID := c.Param("id")
	project, err := h.store.GetProjectByID(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Project not found"}})
		return
	}
	if project.OrgID != c.GetString("org_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Project name required"}})
		return
	}
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	projectID := c.Param("id")
	if err := h.store.UpdateProjectName(c.Request.Context(), projectID, orgID, req.Name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Project not found"}})
		return
	}
	_ = h.store.LogAuditEvent(c.Request.Context(), &userID, orgID, "project.updated", "project", projectID, map[string]any{"name": req.Name})
	c.JSON(http.StatusOK, gin.H{"message": "Project updated"})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	projectID := c.Param("id")
	if err := h.store.DeleteProject(c.Request.Context(), projectID, orgID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Project not found"}})
		return
	}
	_ = h.store.LogAuditEvent(c.Request.Context(), &userID, orgID, "project.deleted", "project", projectID, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}
