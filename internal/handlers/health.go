package handlers

import (
	"net/http"

	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

// Healthz is a simple liveness probe.
func Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz checks database connectivity.
func Readyz(appStore store.AppStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify we can actually reach the database.
		if _, err := appStore.CountProjects(c.Request.Context(), "__readyz_probe__"); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "database unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
