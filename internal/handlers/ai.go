package handlers

import (
	"io"
	"net/http"

	"entsaas/internal/ai"
	"entsaas/internal/billing"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AIHandler serves AI chat completion endpoints.
type AIHandler struct {
	provider ai.Provider
	store    store.AppStore
}

// NewAIHandler creates a new AI handler. Returns nil if AI is disabled.
func NewAIHandler(cfg ai.Config, s store.AppStore) *AIHandler {
	if !cfg.Enabled {
		return nil
	}
	provider, err := ai.NewProvider(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("AI provider init failed, AI endpoints disabled")
		return nil
	}
	log.Info().Str("provider", cfg.ProviderName).Str("model", cfg.Model).Msg("AI provider initialized")
	return &AIHandler{provider: provider, store: s}
}

// Chat handles POST /v1/ai/chat — streaming SSE chat completion.
func (h *AIHandler) Chat(c *gin.Context) {
	var req struct {
		Messages []ai.Message `json:"messages" binding:"required"`
		Model    string       `json:"model,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "messages array is required"},
		})
		return
	}

	orgID := c.GetString("org_id")

	// Quota/feature check
	enforcer := billing.NewEnforcer(h.store)
	if err := enforcer.CheckFeature(c.Request.Context(), orgID, string(billing.KeyAIAssistantEnabled)); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{"code": "QUOTA_EXCEEDED", "message": err.Error()},
		})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "At least one message is required"},
		})
		return
	}

	// SEC-15: Cap messages to prevent cost explosion via mega-token LLM calls.
	if len(req.Messages) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "Maximum 100 messages per request"},
		})
		return
	}

	userID := c.GetString("user_id")

	// Audit log the AI interaction.
	_ = h.store.LogAuditEvent(c.Request.Context(), &userID, orgID,
		"ai.chat", "ai", "chat", map[string]any{
			"model":         h.provider.ModelName(),
			"message_count": len(req.Messages),
		})

	// SSE streaming response.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	c.Stream(func(w io.Writer) bool {
		err := h.provider.Stream(c.Request.Context(), ai.CompletionRequest{
			Messages: req.Messages,
			Model:    req.Model,
			Stream:   true,
		}, func(chunk ai.StreamChunk) error {
			if chunk.Done {
				c.SSEvent("message", "[DONE]")
			} else {
				c.SSEvent("message", chunk.Content)
			}
			return nil
		})

		if err != nil {
			log.Error().Err(err).Msg("AI stream error")
			// SEC-18: Don't leak upstream provider error details to the client.
			c.SSEvent("error", "AI completion encountered an error")
		}

		return false // stop streaming
	})
}

// Complete handles POST /v1/ai/complete — single (non-streaming) completion.
func (h *AIHandler) Complete(c *gin.Context) {
	var req struct {
		Messages []ai.Message `json:"messages" binding:"required"`
		Model    string       `json:"model,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "messages array is required"},
		})
		return
	}

	orgID := c.GetString("org_id")

	// Quota/feature check
	enforcer := billing.NewEnforcer(h.store)
	if err := enforcer.CheckFeature(c.Request.Context(), orgID, string(billing.KeyAIAssistantEnabled)); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{"code": "QUOTA_EXCEEDED", "message": err.Error()},
		})
		return
	}

	// SEC-15: Cap messages to prevent cost explosion via mega-token LLM calls.
	if len(req.Messages) == 0 || len(req.Messages) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "messages array must contain 1-100 entries"},
		})
		return
	}

	userID := c.GetString("user_id")

	resp, err := h.provider.Complete(c.Request.Context(), ai.CompletionRequest{
		Messages: req.Messages,
		Model:    req.Model,
	})
	if err != nil {
		log.Error().Err(err).Msg("AI completion error")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "AI_ERROR", "message": "AI completion failed"},
		})
		return
	}

	// Audit log.
	_ = h.store.LogAuditEvent(c.Request.Context(), &userID, orgID,
		"ai.complete", "ai", "completion", map[string]any{
			"model":         resp.Model,
			"prompt_tokens": resp.PromptTokens,
			"output_tokens": resp.OutputTokens,
		})

	c.JSON(http.StatusOK, gin.H{
		"content":       resp.Content,
		"model":         resp.Model,
		"prompt_tokens": resp.PromptTokens,
		"output_tokens": resp.OutputTokens,
		"finish_reason": resp.FinishReason,
	})
}
