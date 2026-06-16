package billing

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// EntitlementKey represents a unique identifier for plan limits or feature gates.
type EntitlementKey string

const (
	// Capacity entitlements
	KeyMaxProjects           EntitlementKey = "max_projects"
	KeyMaxMembers            EntitlementKey = "max_members"
	KeyAuditLogRetentionDays EntitlementKey = "audit_log_retention_days"

	// Feature entitlements
	KeyAIAssistantEnabled    EntitlementKey = "ai_assistant_enabled"
	KeySSOEnabled            EntitlementKey = "sso_enabled"
	KeyPrioritySupport       EntitlementKey = "priority_support"
)

// Entitlements encapsulates the configured limits and features for a plan or subscription.
type Entitlements struct {
	MaxProjects           int64 `yaml:"max_projects" json:"max_projects"`
	MaxMembers            int64 `yaml:"max_members" json:"max_members"`
	AIAssistantEnabled    bool  `yaml:"ai_assistant_enabled" json:"ai_assistant_enabled"`
	SSOEnabled            bool  `yaml:"sso_enabled" json:"sso_enabled"`
	PrioritySupport       bool  `yaml:"priority_support" json:"priority_support"`
	AuditLogRetentionDays int64 `yaml:"audit_log_retention_days" json:"audit_log_retention_days"`
}

// PlanDefinition maps to a tier in billing.yaml
type PlanDefinition struct {
	ID            string       `yaml:"id" json:"id"`
	Name          string       `yaml:"name" json:"name"`
	Description   string       `yaml:"description" json:"description"`
	PriceMonthly  float64      `yaml:"price_monthly" json:"price_monthly"`
	PriceYearly   float64      `yaml:"price_yearly" json:"price_yearly"`
	StripePriceID string       `yaml:"stripe_price_id" json:"stripe_price_id"`
	IsSelfServe   bool         `yaml:"is_self_serve" json:"is_self_serve"`
	Features      Entitlements `yaml:"features" json:"-"`
	Capacity      Entitlements `yaml:"capacity" json:"-"`
}

// EffectiveEntitlements merges features and capacity back into a flat map or struct for the enforcer.
// We keep the Entitlements struct as a flat return for GetEffectiveEntitlements.
func (p *PlanDefinition) EffectiveEntitlements() Entitlements {
	return Entitlements{
		MaxProjects:           p.Capacity.MaxProjects,
		MaxMembers:            p.Capacity.MaxMembers,
		AuditLogRetentionDays: p.Capacity.AuditLogRetentionDays,
		AIAssistantEnabled:    p.Features.AIAssistantEnabled,
		SSOEnabled:            p.Features.SSOEnabled,
		PrioritySupport:       p.Features.PrioritySupport,
	}
}

// CatalogConfig is the root struct for billing.yaml
type CatalogConfig struct {
	Provider       string           `yaml:"provider"`
	CancelBehavior string           `yaml:"cancel_behavior"`
	Currency       string           `yaml:"currency"`
	Plans          []PlanDefinition `yaml:"plans"`
}

var GlobalCatalog *CatalogConfig

// LoadConfig parses the YAML file at the given path into GlobalCatalog
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read billing config %q: %w", path, err)
	}

	var cfg CatalogConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse billing config %q: %w", path, err)
	}

	GlobalCatalog = &cfg
	return nil
}

// GetPlan returns the specific plan from the catalog
func GetPlan(slug string) (*PlanDefinition, bool) {
	if GlobalCatalog == nil {
		return nil, false
	}
	for _, p := range GlobalCatalog.Plans {
		if p.ID == slug {
			return &p, true
		}
	}
	return nil, false
}

// FreePlanEntitlements returns free-plan entitlements from the in-memory catalog
// in the (slug, map[string]any) format expected by store.GetEffectiveEntitlements.
// Used as a last-resort fallback when neither subscription nor plans table row exists.
func FreePlanEntitlements() (string, map[string]any, error) {
	plan, ok := GetPlan("free")
	if !ok {
		return "", nil, fmt.Errorf("billing: free plan not found in catalog")
	}
	ents := plan.EffectiveEntitlements()
	m := map[string]any{
		string(KeyMaxProjects):           ents.MaxProjects,
		string(KeyMaxMembers):            ents.MaxMembers,
		string(KeyAuditLogRetentionDays): ents.AuditLogRetentionDays,
		string(KeyAIAssistantEnabled):    ents.AIAssistantEnabled,
		string(KeySSOEnabled):            ents.SSOEnabled,
		string(KeyPrioritySupport):       ents.PrioritySupport,
	}
	return "free", m, nil
}

// CancelBehavior returns the configured cancellation style from the catalog.
// Returns "period_end" (default) or "immediate".
func CancelBehavior() string {
	if GlobalCatalog != nil && GlobalCatalog.CancelBehavior != "" {
		return GlobalCatalog.CancelBehavior
	}
	return "period_end"
}

// Currency returns the configured default currency from the catalog.
func Currency() string {
	if GlobalCatalog != nil && GlobalCatalog.Currency != "" {
		return GlobalCatalog.Currency
	}
	return "USD"
}

