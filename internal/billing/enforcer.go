package billing

import (
	"context"
	"fmt"

	"entsaas/internal/store"
)

// QuotaExceededError is returned when a numeric limit is reached or exceeded.
type QuotaExceededError struct {
	Dimension   string `json:"dimension"`
	Limit       int64  `json:"limit"`
	Current     int64  `json:"current"`
	UpgradePath string `json:"upgrade_path"`
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("quota exceeded for %s: limit %d, current %d. %s", e.Dimension, e.Limit, e.Current, e.UpgradePath)
}

// FeatureUnavailableError is returned when a gated feature flag is inactive on their plan.
type FeatureUnavailableError struct {
	Feature     string `json:"feature"`
	Plan        string `json:"plan"`
	UpgradePath string `json:"upgrade_path"`
}

func (e *FeatureUnavailableError) Error() string {
	return fmt.Sprintf("feature %s is not available on %s plan. %s", e.Feature, e.Plan, e.UpgradePath)
}

// Enforcer performs active limit/feature gate checking.
type Enforcer struct {
	store store.AppStore
}

// NewEnforcer instantiates a new quota enforcer.
func NewEnforcer(s store.AppStore) *Enforcer {
	return &Enforcer{store: s}
}

// CheckProjectLimit verifies that the organization is allowed to create another project.
func (e *Enforcer) CheckProjectLimit(ctx context.Context, orgID string) error {
	_, ents, err := e.store.GetEffectiveEntitlements(ctx, orgID)
	if err != nil {
		return err
	}

	limitVal, ok := ents[string(KeyMaxProjects)]
	if !ok {
		return nil // fail-open if not configured
	}

	limit := toInt64(limitVal)
	if limit < 0 {
		return nil // unlimited
	}

	current, err := e.store.CountProjects(ctx, orgID)
	if err != nil {
		return err
	}

	if int64(current) >= limit {
		return &QuotaExceededError{
			Dimension:   string(KeyMaxProjects),
			Limit:       limit,
			Current:     int64(current),
			UpgradePath: "Upgrade to Pro or Enterprise for unlimited projects.",
		}
	}

	return nil
}

// CheckMemberLimit verifies that the organization is allowed to add another member.
func (e *Enforcer) CheckMemberLimit(ctx context.Context, orgID string) error {
	_, ents, err := e.store.GetEffectiveEntitlements(ctx, orgID)
	if err != nil {
		return err
	}

	limitVal, ok := ents[string(KeyMaxMembers)]
	if !ok {
		return nil // fail-open if not configured
	}

	limit := toInt64(limitVal)
	if limit < 0 {
		return nil // unlimited
	}

	current, err := e.store.CountUsers(ctx, orgID)
	if err != nil {
		return err
	}

	if int64(current) >= limit {
		return &QuotaExceededError{
			Dimension:   string(KeyMaxMembers),
			Limit:       limit,
			Current:     int64(current),
			UpgradePath: "Upgrade to Pro or Enterprise to add more members.",
		}
	}

	return nil
}

// CheckFeature verifies that a feature gate is enabled on the current plan.
func (e *Enforcer) CheckFeature(ctx context.Context, orgID string, feature string) error {
	slug, ents, err := e.store.GetEffectiveEntitlements(ctx, orgID)
	if err != nil {
		return err
	}

	val, ok := ents[feature]
	if !ok {
		return &FeatureUnavailableError{
			Feature:     feature,
			Plan:        slug,
			UpgradePath: fmt.Sprintf("Upgrade to a plan that includes %s.", feature),
		}
	}

	enabled, _ := val.(bool)
	if !enabled {
		return &FeatureUnavailableError{
			Feature:     feature,
			Plan:        slug,
			UpgradePath: fmt.Sprintf("Upgrade to a plan that includes %s.", feature),
		}
	}

	return nil
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}
