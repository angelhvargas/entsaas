package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// SyncBillingPlan upserts a plan and its entitlements as a new version if they changed.
func (s *PostgresStore) SyncBillingPlan(ctx context.Context, slug, name, description string, entitlements map[string]any, isSelfServe bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Upsert plan
	var planID string
	err = tx.QueryRow(ctx, `
		INSERT INTO plans (slug, display_name, description, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (slug) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			is_active = true
		RETURNING id
	`, slug, name, description).Scan(&planID)
	if err != nil {
		return fmt.Errorf("upsert plan %s: %w", slug, err)
	}

	entBytes, err := json.Marshal(entitlements)
	if err != nil {
		return fmt.Errorf("marshal entitlements: %w", err)
	}

	// Check if latest version has different entitlements
	var currentEntBytes []byte
	var currentVersion int
	err = tx.QueryRow(ctx, `
		SELECT version, entitlements FROM plan_versions
		WHERE plan_id = $1
		ORDER BY version DESC LIMIT 1
	`, planID).Scan(&currentVersion, &currentEntBytes)

	insertNewVersion := false
	if err != nil {
		// no rows, insert version 1
		currentVersion = 0
		insertNewVersion = true
	} else {
		// Compare JSON. This is simple string comparison because we generated both from Go map serialization.
		if string(currentEntBytes) != string(entBytes) {
			insertNewVersion = true
		}
	}

	if insertNewVersion {
		newVer := currentVersion + 1
		notes := fmt.Sprintf("Synced from config to version %d", newVer)
		_, err = tx.Exec(ctx, `
			INSERT INTO plan_versions (plan_id, version, entitlements, notes)
			VALUES ($1, $2, $3, $4)
		`, planID, newVer, entBytes, notes)
		if err != nil {
			return fmt.Errorf("insert plan version: %w", err)
		}
	}

	return tx.Commit(ctx)
}
