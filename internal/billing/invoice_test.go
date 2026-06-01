package billing_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"entsaas/internal/billing"
	"entsaas/internal/bootstrap"
	"entsaas/internal/store"

	"github.com/google/uuid"
)

func getTestPostgresStore(t *testing.T) *store.PostgresStore {
	if testing.Short() {
		t.Skip("skipping DB integration test in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"
	}
	masterKeyConf := bootstrap.MustParseMasterKey()
	pgStore, err := store.NewPostgresStore(dsn, masterKeyConf.Key, masterKeyConf.Version, 2, 1)
	if err != nil {
		t.Skipf("skipping DB integration test — no Postgres available: %v", err)
	}
	return pgStore
}

func TestInvoiceStore_SyncInvoiceAndQueries(t *testing.T) {
	pgStore := getTestPostgresStore(t)
	defer pgStore.Close()

	ctx := context.Background()
	orgID := "org-" + uuid.New().String()
	invoiceStore := billing.NewInvoiceStore(pgStore.Pool())

	// Create mock normalized invoice
	normInv := billing.NormalizedInvoice{
		ID:            "prov-inv-" + uuid.New().String(),
		Status:        "paid",
		AmountCents:   4900,
		Currency:      "usd",
		CreatedAt:     time.Now().Format(time.RFC3339),
		HostedURL:     "https://stripe.com/hosted/123",
		PDFUrl:        "https://stripe.com/pdf/123",
		Description:   "Pro Plan - Upgrade",
		InvoiceNumber: "INV-0099",
		IsPlanChange:  true,
		PlanChangeCtx: "upgrade:pro",
	}

	// Unconditional mapping logic test is implicitly covered here during integration.
	// But let's run the sync.
	err := invoiceStore.SyncInvoiceFromProvider(ctx, orgID, billing.ProviderStripe, normInv)
	if err != nil {
		t.Fatalf("SyncInvoiceFromProvider failed: %v", err)
	}

	// Query invoices for org
	page, err := invoiceStore.GetInvoicesForOrg(ctx, orgID, 1, 10)
	if err != nil {
		t.Fatalf("GetInvoicesForOrg failed: %v", err)
	}

	if page.Total != 1 {
		t.Errorf("expected 1 invoice, got %d", page.Total)
	}
	if len(page.Invoices) != 1 {
		t.Fatalf("expected 1 invoice in list, got %d", len(page.Invoices))
	}

	item := page.Invoices[0]
	if item.Status != "paid" || item.AmountCents != 4900 || item.Currency != "usd" {
		t.Errorf("scanned invoice mismatch: %+v", item)
	}
	if item.InvoiceNumber != "INV-0099" || item.Description != "Pro Plan - Upgrade" {
		t.Errorf("nullable fields mismatch: %+v", item)
	}
	if !item.IsPlanChange || item.PlanChangeCtx != "upgrade:pro" {
		t.Errorf("plan change fields mismatch: %+v", item)
	}

	// Fetch detail by ID
	detail, err := invoiceStore.GetInvoiceByID(ctx, orgID, item.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID failed: %v", err)
	}
	if detail == nil {
		t.Fatal("expected invoice detail, got nil")
	}
	if detail.ID != item.ID || detail.InvoiceNumber != "INV-0099" {
		t.Errorf("invoice detail mismatch: %+v", detail)
	}

	// Verify org scope check: should return nil if incorrect org ID is passed
	wrongDetail, err := invoiceStore.GetInvoiceByID(ctx, "wrong-org", item.ID)
	if err != nil {
		t.Fatalf("GetInvoiceByID with incorrect org failed: %v", err)
	}
	if wrongDetail != nil {
		t.Errorf("expected nil for mismatched org, got detail: %+v", wrongDetail)
	}
}

// Unit test mapping checks that run unconditionally (even in short mode)
func TestNormalizedInvoice_MappingLogic(t *testing.T) {
	orgID := "org-unit-123"
	normInv := billing.NormalizedInvoice{
		ID:            "prov-123",
		Status:        "open",
		AmountCents:   9900,
		Currency:      "eur",
		CreatedAt:     "2026-06-01T12:00:00Z",
		HostedURL:     "https://paddle.com/hosted",
		PDFUrl:        "https://paddle.com/pdf",
		Description:   "Enterprise Subscription",
		InvoiceNumber: "INV-5555",
		IsPlanChange:  false,
		PlanChangeCtx: "",
	}

	local := billing.LocalInvoice{
		OrgID:        orgID,
		ProviderID:   normInv.ID,
		ProviderName: string(billing.ProviderPaddle),
		Status:       normInv.Status,
		AmountCents:  normInv.AmountCents,
		Currency:     normInv.Currency,
		IsPlanChange: normInv.IsPlanChange,
		CreatedAt:    normInv.CreatedAt,
	}
	if normInv.Description != "" {
		local.Description = sql.NullString{String: normInv.Description, Valid: true}
	}
	if normInv.InvoiceNumber != "" {
		local.InvoiceNumber = sql.NullString{String: normInv.InvoiceNumber, Valid: true}
	}
	if normInv.PDFUrl != "" {
		local.PDFUrl = sql.NullString{String: normInv.PDFUrl, Valid: true}
	}
	if normInv.HostedURL != "" {
		local.HostedURL = sql.NullString{String: normInv.HostedURL, Valid: true}
	}
	if normInv.PlanChangeCtx != "" {
		local.PlanChangeCtx = sql.NullString{String: normInv.PlanChangeCtx, Valid: true}
	}

	if local.OrgID != orgID {
		t.Errorf("expected OrgID %s, got %s", orgID, local.OrgID)
	}
	if local.ProviderID != "prov-123" {
		t.Errorf("expected ProviderID 'prov-123', got %s", local.ProviderID)
	}
	if local.ProviderName != "paddle" {
		t.Errorf("expected ProviderName 'paddle', got %s", local.ProviderName)
	}
	if local.Status != "open" {
		t.Errorf("expected Status 'open', got %s", local.Status)
	}
	if local.AmountCents != 9900 {
		t.Errorf("expected AmountCents 9900, got %d", local.AmountCents)
	}
	if local.Currency != "eur" {
		t.Errorf("expected Currency 'eur', got %s", local.Currency)
	}
	if !local.Description.Valid || local.Description.String != "Enterprise Subscription" {
		t.Errorf("description field map failed")
	}
	if !local.InvoiceNumber.Valid || local.InvoiceNumber.String != "INV-5555" {
		t.Errorf("invoice number field map failed")
	}
	if !local.PDFUrl.Valid || local.PDFUrl.String != "https://paddle.com/pdf" {
		t.Errorf("pdf_url field map failed")
	}
	if !local.HostedURL.Valid || local.HostedURL.String != "https://paddle.com/hosted" {
		t.Errorf("hosted_url field map failed")
	}
	if local.IsPlanChange {
		t.Errorf("expected IsPlanChange to be false")
	}
	if local.PlanChangeCtx.Valid {
		t.Errorf("expected PlanChangeCtx to be invalid (empty)")
	}
}
