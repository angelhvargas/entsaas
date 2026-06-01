package handlers

import (
	"net/http"
	"os"
	"time"

	"entsaas/internal/billing"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// BillingHandler exposes subscription and checkout endpoints.
type BillingHandler struct {
	provider billing.Provider
	store    store.AppStore
}

func NewBillingHandler(p billing.Provider, s store.AppStore) *BillingHandler {
	return &BillingHandler{provider: p, store: s}
}

// GetPlan returns the organization's effective plan and entitlements.
func (h *BillingHandler) GetPlan(c *gin.Context) {
	orgID := c.GetString("org_id")
	slug, ents, err := h.store.GetEffectiveEntitlements(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to load plan entitlements"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"plan_slug":    slug,
		"entitlements": ents,
	})
}

// GetPlans returns all available billing plans from the configuration catalog.
func (h *BillingHandler) GetPlans(c *gin.Context) {
	if billing.GlobalCatalog == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Catalog not initialized"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": billing.GlobalCatalog.Plans})
}

// GetSubscription returns the current subscription for the caller's org.
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	orgID := c.GetString("org_id")
	sub, err := h.provider.GetSubscription(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to load subscription"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subscription": sub})
}

// CreateCheckout initiates a plan upgrade checkout session.
func (h *BillingHandler) CreateCheckout(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "plan_id is required"}})
		return
	}

	baseURL := os.Getenv("ENTSAAS_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	session, err := h.provider.CreateCheckoutSession(
		c.Request.Context(),
		c.GetString("org_id"),
		req.PlanID,
		baseURL+"/settings/billing/checkout/success",
		baseURL+"/settings/billing/checkout/cancel",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to create checkout session"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": session.CheckoutURL, "session_id": session.SessionID})
}

// CreatePortal returns a billing portal URL so the user can manage their subscription.
func (h *BillingHandler) CreatePortal(c *gin.Context) {
	orgID := c.GetString("org_id")
	ctx := c.Request.Context()

	sub, err := h.store.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("billing: portal session get subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to retrieve subscription info"}})
		return
	}

	custRef := orgID
	if sub != nil {
		if sub.ProviderCustomerID != "" {
			custRef = sub.ProviderCustomerID
		} else if sub.ProviderSubscriptionID != "" {
			custRef = sub.ProviderSubscriptionID
		}
	}

	portal, err := h.provider.CreatePortalSession(ctx, custRef)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("billing: failed to create portal session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to create portal session"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"portal_url": portal.PortalURL})
}

// Webhook receives and processes inbound billing webhooks.
// The route must be registered WITHOUT auth middleware.
func (h *BillingHandler) Webhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	// Build header map for signature verification.
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	event := &billing.WebhookEvent{
		Type:    c.GetHeader("Stripe-Event") + c.GetHeader("X-Paddle-Event"), // adapter-specific
		Payload: map[string]any{},
		Raw:     body,
		Headers: headers,
	}

	result, err := h.provider.HandleWebhook(c.Request.Context(), event)
	if err != nil {
		// SEC-11: Log the real error but don't expose internal details to callers.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Webhook processing failed"})
		return
	}

	// Persist subscription state if the webhook carries actionable data.
	if result.OrgID != "" && result.Status != "" {
		// Resolve plan ID from slug if available, otherwise use existing subscription's plan.
		planID := ""
		planVersionID := ""
		if result.PlanSlug != "" {
			planID, planVersionID = h.resolvePlanIDs(c, result.PlanSlug)
		}

		// Try to find existing subscription to merge data.
		existing, _ := h.store.GetSubscriptionByOrgID(c.Request.Context(), result.OrgID)
		if existing != nil {
			// Merge: keep existing plan if webhook didn't specify one.
			if planID == "" {
				planID = existing.PlanID
				planVersionID = existing.PlanVersionID
			}
			existing.Status = result.Status
			existing.ProviderSubscriptionID = result.ProviderSubscriptionID
			existing.ProviderCustomerID = result.ProviderCustomerID
			existing.CancelAtPeriodEnd = result.CancelAtPeriodEnd
			if result.CurrentPeriodStart != nil {
				existing.CurrentPeriodStart = result.CurrentPeriodStart
			}
			if result.CurrentPeriodEnd != nil {
				existing.CurrentPeriodEnd = result.CurrentPeriodEnd
			}
			if result.Status == "canceled" {
				now := time.Now()
				existing.CanceledAt = &now
			}
			existing.PlanID = planID
			existing.PlanVersionID = planVersionID
			if err := h.store.UpdateSubscription(c.Request.Context(), existing); err != nil {
				log.Error().Err(err).Str("org_id", result.OrgID).Msg("SEC-07: failed to update subscription")
			}
		} else {
			// Create new subscription.
			sub := &store.Subscription{
				OrgID:                  result.OrgID,
				PlanID:                 planID,
				PlanVersionID:          planVersionID,
				Status:                 result.Status,
				ProviderSubscriptionID: result.ProviderSubscriptionID,
				ProviderCustomerID:     result.ProviderCustomerID,
				CurrentPeriodStart:     result.CurrentPeriodStart,
				CurrentPeriodEnd:       result.CurrentPeriodEnd,
				CancelAtPeriodEnd:      result.CancelAtPeriodEnd,
			}
			if err := h.store.CreateSubscription(c.Request.Context(), sub); err != nil {
				log.Error().Err(err).Str("org_id", result.OrgID).Msg("SEC-07: failed to create subscription")
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": result.EventType})
}

// resolvePlanIDs looks up internal plan_id and latest plan_version_id for a plan slug.
func (h *BillingHandler) resolvePlanIDs(c *gin.Context, slug string) (string, string) {
	var planID, versionID string
	// Use the store's entitlements query pattern but just get the IDs.
	row := h.store.(*store.PostgresStore).Pool().QueryRow(c.Request.Context(), `
		SELECT p.id, COALESCE(pv.id::text, '')
		FROM plans p
		LEFT JOIN plan_versions pv ON pv.plan_id = p.id
		WHERE p.slug = $1
		ORDER BY pv.version DESC
		LIMIT 1
	`, slug)
	_ = row.Scan(&planID, &versionID)
	return planID, versionID
}

