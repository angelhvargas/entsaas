package handlers

import (
	"encoding/json"
	"net/http"

	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

type PrefsHandler struct{ store store.AppStore }

func NewPrefsHandler(s store.AppStore) *PrefsHandler { return &PrefsHandler{store: s} }

func (h *PrefsHandler) Get(c *gin.Context) {
	prefs, err := h.store.GetUserPreferences(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to fetch preferences"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

func (h *PrefsHandler) Set(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Invalid preferences"}})
		return
	}

	// SEC-16: Limit preferences payload size to prevent storage DoS.
	raw, _ := json.Marshal(req)
	if len(raw) > 16*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "Preferences payload too large (max 16KB)"}})
		return
	}

	if err := h.store.SetUserPreferences(c.Request.Context(), c.GetString("user_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to save preferences"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Preferences saved"})
}
