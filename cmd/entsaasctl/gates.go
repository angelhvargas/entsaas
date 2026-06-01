package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"entsaas/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

type GateOptions struct {
	Action      string
	Risk        RiskLevel
	OrgID       string
	UserID      string
	Reason      string
	DryRun      bool
	YesFlag     bool
	TimeoutFlag string
	Metadata    map[string]any
}

// GetActorString returns the validated actor in the format "entsaasctl/<os_user>@<hostname>"
func GetActorString() string {
	osUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = osUser.Username
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("entsaasctl/%s@%s", username, hostname)
}

// CheckPromptConfirmation handles interactive safety gates.
func CheckPromptConfirmation(opts GateOptions) (bool, error) {
	// 1. Check unattended bypass (belt-and-suspenders)
	allowUnattended := os.Getenv("ENTSAASCTL_ALLOW_UNATTENDED") == "true"
	if opts.YesFlag && allowUnattended {
		return true, nil
	}

	// If interactive bypass is partially set (one but not both), warn and refuse or continue to prompt
	if opts.YesFlag && !allowUnattended {
		return false, fmt.Errorf("Gate 3 Rejected: --yes passed but ENTSAASCTL_ALLOW_UNATTENDED=true is NOT set")
	}
	if !opts.YesFlag && allowUnattended {
		// Proceed to standard TTY prompt
	}

	// 2. Check if we have a TTY for interactive prompt
	// In Go, we can check if Stdin is a character device (TTY).
	fileInfo, err := os.Stdin.Stat()
	if err != nil || (fileInfo.Mode()&os.ModeCharDevice) == 0 {
		return false, errors.New("Gate 3 Rejected: not a TTY and unattended bypass flags are not fully set")
	}

	// 3. Prompt user based on risk level
	reader := bufio.NewReader(os.Stdin)
	if opts.Risk == RiskCritical {
		fmt.Printf("⚠️  WARNING: You are about to perform a CRITICAL, potentially destructive operation: %s\n", opts.Action)
		fmt.Print("Type \"YES\" to confirm: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		text = strings.TrimSpace(text)
		if text == "YES" {
			return false, nil // proceed
		}
		return true, errors.New("Gate 3 Rejected: Confirmation failed (must type exact word 'YES')")
	} else if opts.Risk == RiskMedium || opts.Risk == RiskHigh {
		fmt.Printf("⚠️  WARNING: You are about to perform a %s operation: %s\n", opts.Risk, opts.Action)
		fmt.Print("Proceed? [y/N]: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		text = strings.TrimSpace(strings.ToLower(text))
		if text == "y" || text == "yes" {
			return false, nil // proceed
		}
		return true, errors.New("Gate 3 Rejected: Operation cancelled by user")
	}

	return false, nil // Low risk, no prompt
}

// ExecuteWithGate executes a database mutation within the safety gates pipeline
func ExecuteWithGate(ctx context.Context, pgStore *store.PostgresStore, opts GateOptions, mutateFunc func(ctx context.Context, tx pgx.Tx) error) error {
	startTime := time.Now()

	// Gate 1: Input Validation
	if opts.Action == "" {
		return errors.New("Gate 1 Rejected: Action must be specified")
	}
	if (opts.Risk == RiskHigh || opts.Risk == RiskCritical) && opts.Reason == "" {
		return fmt.Errorf("Gate 1 Rejected: Reason (--reason) is mandatory for %s operations", opts.Risk)
	}

	// Resolve actual Org ID from slug if it's the default or fallback
	orgID := opts.OrgID
	if orgID == "" {
		// Attempt to fetch first organization
		var firstID string
		err := pgStore.Pool().QueryRow(ctx, "SELECT id FROM organizations LIMIT 1").Scan(&firstID)
		if err == nil {
			orgID = firstID
		} else {
			// fallback placeholder
			orgID = "00000000-0000-0000-0000-000000000000"
		}
	}

	// Gate 2: Dry Run Check
	if opts.DryRun {
		fmt.Printf("[DRY-RUN] Action: %s\n", opts.Action)
		fmt.Printf("[DRY-RUN] Risk Level: %s\n", opts.Risk)
		fmt.Printf("[DRY-RUN] Target Org: %s\n", orgID)
		if opts.UserID != "" {
			fmt.Printf("[DRY-RUN] Target User: %s\n", opts.UserID)
		}
		if opts.Reason != "" {
			fmt.Printf("[DRY-RUN] Reason: %s\n", opts.Reason)
		}
		fmt.Printf("[DRY-RUN] Preview complete. Exiting without changes.\n")
		// Write a dry-run log to stderr/stdout and return specific exit code
		os.Exit(3)
	}

	// Gate 3: Interactive Confirmation
	unattendedBypass, err := CheckPromptConfirmation(opts)
	if err != nil {
		return err
	}

	// Gate 4: Execution Timeout setup
	defaultTimeout := 30 * time.Second
	if opts.TimeoutFlag != "" {
		parsed, err := time.ParseDuration(opts.TimeoutFlag)
		if err == nil {
			defaultTimeout = parsed
		} else {
			return fmt.Errorf("Gate 4 Rejected: invalid timeout duration %q", opts.TimeoutFlag)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Gate 5: Transactional execution and Audit Logging
	tx, err := pgStore.Pool().Begin(timeoutCtx)
	if err != nil {
		return fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(timeoutCtx)

	// Execute actual mutation
	mutateErr := mutateFunc(timeoutCtx, tx)

	// Log audit event within the same transaction to guarantee atomicity
	durationMs := time.Since(startTime).Milliseconds()
	actorStr := GetActorString()

	meta := map[string]any{
		"unattended":  unattendedBypass,
		"duration_ms": durationMs,
		"risk_level":  opts.Risk,
		"reason":      opts.Reason,
		"success":     mutateErr == nil,
	}
	if opts.Metadata != nil {
		for k, v := range opts.Metadata {
			meta[k] = v
		}
	}

	if mutateErr != nil {
		// 1. Rollback transaction immediately to discard any partial mutation edits!
		_ = tx.Rollback(timeoutCtx)

		// 2. Write failed audit log entry outside transaction
		meta["error"] = mutateErr.Error()
		metaJSON, _ := json.Marshal(meta)
		_, _ = pgStore.Pool().Exec(ctx,
			`INSERT INTO audit_log (id, actor_id, org_id, action, entity_type, entity_id, metadata, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
			uuid.New(), actorStr, orgID, "entsaasctl."+opts.Action, "system", orgID, metaJSON)

		return fmt.Errorf("Operation failed: %w", mutateErr)
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte("{}")
	}

	_, auditErr := tx.Exec(timeoutCtx,
		`INSERT INTO audit_log (id, actor_id, org_id, action, entity_type, entity_id, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		uuid.New(), actorStr, orgID, "entsaasctl."+opts.Action, "system", orgID, metaJSON)

	if auditErr != nil {
		_ = tx.Rollback(timeoutCtx)
		return fmt.Errorf("Gate 5 Failed: Failed to write audit log entry. Transaction rolled back. Err: %w", auditErr)
	}

	// Both succeeded: Commit!
	if err := tx.Commit(timeoutCtx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("✅ Success: %s completed successfully (%dms)\n", opts.Action, durationMs)
	return nil
}
