package billing_test

import (
	"os"
	"path/filepath"
	"testing"

	"entsaas/internal/billing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	yamlContent := []byte(`
provider: stripe
cancel_behavior: period_end
currency: USD
plans:
  - id: pro
    name: Pro
    price_monthly: 19
    is_self_serve: true
    features:
      ai_assistant_enabled: true
    capacity:
      max_projects: 5
`)
	cfgPath := filepath.Join(tmpDir, "billing.yaml")
	if err := os.WriteFile(cfgPath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write mock yaml: %v", err)
	}

	if err := billing.LoadConfig(cfgPath); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if billing.GlobalCatalog.Provider != "stripe" {
		t.Errorf("expected provider stripe, got %s", billing.GlobalCatalog.Provider)
	}
	if len(billing.GlobalCatalog.Plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(billing.GlobalCatalog.Plans))
	}

	plan := billing.GlobalCatalog.Plans[0]
	if plan.ID != "pro" {
		t.Errorf("expected plan id 'pro', got %s", plan.ID)
	}
	if !plan.Features.AIAssistantEnabled {
		t.Errorf("expected ai_assistant_enabled to be true")
	}
	if plan.Capacity.MaxProjects != 5 {
		t.Errorf("expected max_projects 5, got %d", plan.Capacity.MaxProjects)
	}

	// Test EffectiveEntitlements
	eff := plan.EffectiveEntitlements()
	if eff.MaxProjects != 5 || !eff.AIAssistantEnabled {
		t.Errorf("EffectiveEntitlements mapping failed")
	}

	// Test GetPlan
	fetchedPlan, ok := billing.GetPlan("pro")
	if !ok || fetchedPlan.Name != "Pro" {
		t.Errorf("GetPlan failed")
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	err := billing.LoadConfig("does-not-exist.yaml")
	if err == nil {
		t.Errorf("expected error when file does not exist")
	}
}
