package handlers

import (
	"net/http"
	"strconv"
	"time"

	"entsaas/internal/billing"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ── Org-Facing Billing Summary structures ────────────────────────────────────

// BillingSubscriptionSummary is the subscription portion of the billing summary.
type BillingSubscriptionSummary struct {
	Status    string `json:"status"`
	PlanSlug  string `json:"plan_slug"`
	PlanName  string `json:"plan_name"`
	StartedAt string `json:"started_at,omitempty"`
}

// PaymentMethodSummary is an advisory display of the payment method on file.
type PaymentMethodSummary struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

// BillingSummaryResponse is the response for GET /v1/billing/summary.
type BillingSummaryResponse struct {
	Subscription    *BillingSubscriptionSummary `json:"subscription"`
	AdvisoryStatus  *string                     `json:"advisory_status"`
	ProviderBacked  bool                        `json:"provider_backed"`
	RenewalDate     *string                     `json:"renewal_date,omitempty"`
	PaymentMethod   *PaymentMethodSummary       `json:"payment_method,omitempty"`
	Invoices        []billing.InvoiceItem       `json:"invoices"`
	ProviderEnabled bool                        `json:"provider_enabled"`
}

// GetBillingSummary handles GET /v1/billing/summary.
// It returns the organization's billing summary including subscription status, advisory
// state, payment method summary, recent invoices, and provider status.
func (h *BillingHandler) GetBillingSummary(c *gin.Context) {
	orgID := c.GetString("org_id")
	ctx := c.Request.Context()

	// Get subscription from DB
	sub, err := h.store.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("billing: failed to load subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to load subscription info"}})
		return
	}

	resp := BillingSummaryResponse{
		ProviderEnabled: true,
		Invoices:        []billing.InvoiceItem{},
	}

	if sub != nil {
		// Resolve Plan name
		planName := sub.PlanSlug
		if plan, ok := billing.GetPlan(sub.PlanSlug); ok {
			planName = plan.Name
		}

		resp.Subscription = &BillingSubscriptionSummary{
			Status:    sub.Status,
			PlanSlug:  sub.PlanSlug,
			PlanName:  planName,
			StartedAt: sub.CreatedAt.Format(time.RFC3339),
		}

		// Dynamic/computed Advisory status
		var adv billing.AdvisoryStatus
		switch sub.Status {
		case "past_due":
			adv = billing.AdvisoryPaymentPastDue
		case "active", "trialing":
			if sub.CancelAtPeriodEnd {
				adv = billing.AdvisoryCancellationPending
			} else {
				adv = billing.AdvisoryPaymentOK
			}
		default:
			adv = billing.AdvisoryPaymentWarning
		}
		advStr := string(adv)
		resp.AdvisoryStatus = &advStr

		resp.ProviderBacked = sub.ProviderSubscriptionID != ""

		if resp.ProviderBacked {
			// Call provider to fetch billing summary
			custRef := sub.ProviderCustomerID
			if custRef == "" {
				custRef = sub.ProviderSubscriptionID
			}

			summary, err := h.provider.FetchBillingSummary(ctx, custRef)
			if err == nil && summary != nil {
				if summary.PaymentMethod != nil {
					resp.PaymentMethod = &PaymentMethodSummary{
						Brand:    summary.PaymentMethod.Brand,
						Last4:    summary.PaymentMethod.Last4,
						ExpMonth: summary.PaymentMethod.ExpMonth,
						ExpYear:  summary.PaymentMethod.ExpYear,
					}
				}
				for _, inv := range summary.Invoices {
					resp.Invoices = append(resp.Invoices, billing.InvoiceItem{
						ID:            inv.ID,
						Status:        inv.Status,
						AmountCents:   inv.AmountCents,
						Currency:      inv.Currency,
						CreatedAt:     inv.CreatedAt,
						HostedURL:     inv.HostedURL,
						PDFUrl:        inv.PDFUrl,
						Description:   inv.Description,
						InvoiceNumber: inv.InvoiceNumber,
						IsPlanChange:  inv.IsPlanChange,
						PlanChangeCtx: inv.PlanChangeCtx,
					})
				}
				if summary.RenewalDate != nil {
					rd := summary.RenewalDate.Format(time.RFC3339)
					resp.RenewalDate = &rd
				}
			} else {
				// Fallback: estimate renewal from current period end
				var renewalDate time.Time
				if sub.CurrentPeriodEnd != nil {
					renewalDate = *sub.CurrentPeriodEnd
				} else {
					renewalDate = estimateRenewalDate(sub.CreatedAt)
				}
				rd := renewalDate.Format(time.RFC3339)
				resp.RenewalDate = &rd
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// GetInvoices handles GET /v1/billing/invoices.
// Returns a paginated list of local stored invoice metadata.
func (h *BillingHandler) GetInvoices(c *gin.Context) {
	orgID := c.GetString("org_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	pgStore, ok := h.store.(*store.PostgresStore)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Invalid store configuration"}})
		return
	}
	invoiceStore := billing.NewInvoiceStore(pgStore.Pool())

	pageData, err := invoiceStore.GetInvoicesForOrg(c.Request.Context(), orgID, page, pageSize)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Msg("billing: failed to load invoices")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to load invoices"}})
		return
	}
	c.JSON(http.StatusOK, pageData)
}

// GetInvoice handles GET /v1/billing/invoices/:id.
// Returns details for a single organization-scoped invoice.
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	orgID := c.GetString("org_id")
	invoiceID := c.Param("id")

	pgStore, ok := h.store.(*store.PostgresStore)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Invalid store configuration"}})
		return
	}
	invoiceStore := billing.NewInvoiceStore(pgStore.Pool())

	invoice, err := invoiceStore.GetInvoiceByID(c.Request.Context(), orgID, invoiceID)
	if err != nil {
		log.Error().Err(err).Str("org_id", orgID).Str("invoice_id", invoiceID).Msg("billing: failed to load invoice detail")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BILLING_ERROR", "message": "Failed to load invoice"}})
		return
	}
	if invoice == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Invoice not found"}})
		return
	}
	c.JSON(http.StatusOK, invoice)
}

// estimateRenewalDate projects the next billing cycle transition from start timestamp.
func estimateRenewalDate(subStart time.Time) time.Time {
	now := time.Now().UTC()
	renewal := subStart
	for renewal.Before(now) {
		renewal = renewal.AddDate(0, 1, 0)
	}
	return renewal
}
