package billing_test

import (
	"testing"

	"entsaas/internal/billing"
)

func TestFreePlanEntitlements_WithCatalog(t *testing.T) {
	// Set up a minimal catalog with a free plan.
	billing.GlobalCatalog = &billing.CatalogConfig{
		Plans: []billing.PlanDefinition{
			{
				ID:   "free",
				Name: "Free",
				Capacity: billing.Entitlements{
					MaxProjects:           1,
					MaxMembers:            1,
					AuditLogRetentionDays: 7,
				},
				Features: billing.Entitlements{
					AIAssistantEnabled: false,
					SSOEnabled:         false,
					PrioritySupport:    false,
				},
			},
		},
	}
	defer func() { billing.GlobalCatalog = nil }()

	slug, ents, err := billing.FreePlanEntitlements()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if slug != "free" {
		t.Errorf("expected slug 'free', got %q", slug)
	}
	if ents == nil {
		t.Fatal("expected entitlements map, got nil")
	}

	// Verify capacity limits.
	if v, ok := ents[string(billing.KeyMaxProjects)]; !ok || v != int64(1) {
		t.Errorf("expected max_projects=1, got %v", v)
	}
	if v, ok := ents[string(billing.KeyMaxMembers)]; !ok || v != int64(1) {
		t.Errorf("expected max_members=1, got %v", v)
	}

	// Verify feature flags are false.
	if v, ok := ents[string(billing.KeyAIAssistantEnabled)]; !ok || v != false {
		t.Errorf("expected ai_assistant_enabled=false, got %v", v)
	}
}

func TestFreePlanEntitlements_NoCatalog(t *testing.T) {
	billing.GlobalCatalog = nil

	_, _, err := billing.FreePlanEntitlements()
	if err == nil {
		t.Fatal("expected error when catalog is nil, got nil")
	}
}

func TestFreePlanEntitlements_NoFreePlan(t *testing.T) {
	billing.GlobalCatalog = &billing.CatalogConfig{
		Plans: []billing.PlanDefinition{
			{ID: "pro", Name: "Pro"},
		},
	}
	defer func() { billing.GlobalCatalog = nil }()

	_, _, err := billing.FreePlanEntitlements()
	if err == nil {
		t.Fatal("expected error when no free plan in catalog, got nil")
	}
}
