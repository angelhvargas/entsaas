package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"entsaas/internal/store"

	"github.com/jackc/pgx/v5"
)

func handleUsersCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("users", flag.ExitOnError)
	orgIDOpt := fs.String("org-id", "", "ID of the organization")
	userIDOpt := fs.String("user-id", "", "ID of the user")
	roleOpt := fs.String("role", "", "Role to assign (e.g. member, admin, owner)")
	activeOpt := fs.String("active", "", "Active status (true/false)")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	switch verb {
	case "list":
		if *orgIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'list'")
			os.Exit(2)
		}
		runUsersList(ctx, pgStore, *orgIDOpt)
	case "set-role":
		if *userIDOpt == "" || *orgIDOpt == "" || *roleOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id, --org-id, and --role are required for 'set-role'")
			os.Exit(2)
		}
		runUsersSetRole(ctx, pgStore, *userIDOpt, *orgIDOpt, *roleOpt)
	case "set-status":
		if *userIDOpt == "" || *orgIDOpt == "" || *activeOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id, --org-id, and --active are required for 'set-status'")
			os.Exit(2)
		}
		activeBool, err := strconv.ParseBool(*activeOpt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid boolean value for --active: %v\n", err)
			os.Exit(2)
		}
		runUsersSetStatus(ctx, pgStore, *userIDOpt, *orgIDOpt, activeBool)
	case "revoke-sessions":
		if *userIDOpt == "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required for 'revoke-sessions'")
			os.Exit(2)
		}
		runUsersRevokeSessions(ctx, pgStore, *userIDOpt)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'users'\n", verb)
		os.Exit(2)
	}
}

func runUsersList(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	users, err := pgStore.GetUsersByOrg(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch users: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tROLE\tACTIVE\tCREATED AT")

	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", u.ID, u.Email, u.Role, u.IsActive, u.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	_ = w.Flush()
	if len(users) == 0 {
		fmt.Println("No users found for this organization.")
	}
}

func runUsersSetRole(ctx context.Context, pgStore *store.PostgresStore, userID, orgID, role string) {
	opts := GateOptions{
		Action:      "users.set-role",
		Risk:        RiskMedium,
		OrgID:       orgID,
		UserID:      userID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
		Metadata:    map[string]any{"role": role},
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		res, err := tx.Exec(ctx,
			`UPDATE users SET role = $3 WHERE id = $1 AND org_id = $2`,
			userID, orgID, role)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return store.ErrNotFound
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to set user role: %v\n", err)
		os.Exit(1)
	}
}

func runUsersSetStatus(ctx context.Context, pgStore *store.PostgresStore, userID, orgID string, active bool) {
	opts := GateOptions{
		Action:      "users.set-status",
		Risk:        RiskMedium,
		OrgID:       orgID,
		UserID:      userID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
		Metadata:    map[string]any{"active": active},
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		res, err := tx.Exec(ctx,
			`UPDATE users SET is_active = $3 WHERE id = $1 AND org_id = $2`,
			userID, orgID, active)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return store.ErrNotFound
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to set user status: %v\n", err)
		os.Exit(1)
	}
}

func runUsersRevokeSessions(ctx context.Context, pgStore *store.PostgresStore, userID string) {
	opts := GateOptions{
		Action:      "users.revoke-sessions",
		Risk:        RiskMedium,
		UserID:      userID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		// Verify user exists
		var userExists bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists)
		if err != nil {
			return err
		}
		if !userExists {
			return store.ErrNotFound
		}

		// Revoke refresh tokens
		_, err = tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
			userID)
		return err
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to revoke sessions: %v\n", err)
		os.Exit(1)
	}
}
