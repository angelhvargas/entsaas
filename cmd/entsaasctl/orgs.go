package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"entsaas/internal/store"

	"github.com/jackc/pgx/v5"
)

func handleOrgsCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("orgs", flag.ExitOnError)
	orgIDOpt := fs.String("org-id", "", "ID of the organization")
	var immediateOpt bool
	fs.BoolVar(&immediateOpt, "immediate", false, "By-pass cooling-off and delete immediately")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	switch verb {
	case "list":
		runOrgsList(ctx, pgStore)
	case "inspect":
		if *orgIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'inspect'")
			os.Exit(2)
		}
		runOrgsInspect(ctx, pgStore, *orgIDOpt)
	case "suspend":
		if *orgIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'suspend'")
			os.Exit(2)
		}
		runOrgsSuspend(ctx, pgStore, *orgIDOpt)
	case "unsuspend":
		if *orgIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'unsuspend'")
			os.Exit(2)
		}
		runOrgsUnsuspend(ctx, pgStore, *orgIDOpt)
	case "delete":
		if *orgIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'delete'")
			os.Exit(2)
		}
		runOrgsDelete(ctx, pgStore, *orgIDOpt, immediateOpt)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'orgs'\n", verb)
		os.Exit(2)
	}
}

func runOrgsList(ctx context.Context, pgStore *store.PostgresStore) {
	rows, err := pgStore.Pool().Query(ctx,
		`SELECT id, name, slug, created_at, suspended_at, deletion_scheduled_at FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: List failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tSTATUS\tDELETION SCHEDULED")

	count := 0
	for rows.Next() {
		var id, name, slug string
		var createdAt time.Time
		var suspendedAt, deletionScheduledAt *time.Time
		if err := rows.Scan(&id, &name, &slug, &createdAt, &suspendedAt, &deletionScheduledAt); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
			os.Exit(1)
		}

		status := "Active"
		if suspendedAt != nil {
			status = "Suspended"
		}
		delSched := "-"
		if deletionScheduledAt != nil {
			delSched = deletionScheduledAt.Format(time.RFC3339)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, name, slug, status, delSched)
		count++
	}

	_ = w.Flush()
	if count == 0 {
		fmt.Println("No organizations found.")
	}
}

func runOrgsInspect(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	org, err := pgStore.GetOrganizationByID(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Organization %s not found: %v\n", orgID, err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(org); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Serialization failed: %v\n", err)
		os.Exit(1)
	}
}

func runOrgsSuspend(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	opts := GateOptions{
		Action:      "orgs.suspend",
		Risk:        RiskHigh,
		OrgID:       orgID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		res, err := tx.Exec(ctx,
			`UPDATE organizations SET suspended_at = NOW(), suspended_reason = $2 WHERE id = $1`,
			orgID, opts.Reason)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return store.ErrNotFound
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Suspension failed: %v\n", err)
		os.Exit(1)
	}
}

func runOrgsUnsuspend(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	opts := GateOptions{
		Action:      "orgs.unsuspend",
		Risk:        RiskMedium,
		OrgID:       orgID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		res, err := tx.Exec(ctx,
			`UPDATE organizations SET suspended_at = NULL, suspended_reason = NULL WHERE id = $1`,
			orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return store.ErrNotFound
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Un-suspension failed: %v\n", err)
		os.Exit(1)
	}
}

func runOrgsDelete(ctx context.Context, pgStore *store.PostgresStore, orgID string, immediate bool) {
	if immediate {
		allowImmediate := os.Getenv("ENTSAASCTL_ALLOW_IMMEDIATE_DELETE") == "true"
		if !allowImmediate {
			fmt.Fprintln(os.Stderr, "Error: Immediate deletion refused. Set ENTSAASCTL_ALLOW_IMMEDIATE_DELETE=true to override.")
			os.Exit(1)
		}

		opts := GateOptions{
			Action:      "orgs.delete-immediate",
			Risk:        RiskCritical,
			OrgID:       orgID,
			Reason:      reasonFlag,
			DryRun:      dryRunFlag,
			YesFlag:     yesFlag,
			TimeoutFlag: timeoutFlag,
			Metadata:    map[string]any{"immediate": true},
		}

		err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
			// Cascading transactional purge
			_, err := tx.Exec(ctx, `DELETE FROM reset_tokens WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM verification_tokens WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM user_credentials WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM user_preferences WHERE user_id IN (SELECT id FROM users WHERE org_id = $1)`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE org_id = $1`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM invites WHERE org_id = $1`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM projects WHERE org_id = $1`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM users WHERE org_id = $1`, orgID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `DELETE FROM audit_log WHERE org_id = $1`, orgID)
			if err != nil {
				return err
			}
			res, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID)
			if err != nil {
				return err
			}
			if res.RowsAffected() == 0 {
				return store.ErrNotFound
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Immediate deletion failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Schedule cooling-off deletion (72h)
		opts := GateOptions{
			Action:      "orgs.delete-scheduled",
			Risk:        RiskCritical,
			OrgID:       orgID,
			Reason:      reasonFlag,
			DryRun:      dryRunFlag,
			YesFlag:     yesFlag,
			TimeoutFlag: timeoutFlag,
			Metadata:    map[string]any{"immediate": false},
		}

		err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
			scheduledTime := time.Now().Add(72 * time.Hour)
			res, err := tx.Exec(ctx,
				`UPDATE organizations SET deletion_scheduled_at = $2 WHERE id = $1`,
				orgID, scheduledTime)
			if err != nil {
				return err
			}
			if res.RowsAffected() == 0 {
				return store.ErrNotFound
			}
			fmt.Printf("⏰ Organization marked for deletion at: %s (72 hours cooling-off period)\n", scheduledTime.Format(time.RFC3339))
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Scheduling deletion failed: %v\n", err)
			os.Exit(1)
		}
	}
}
