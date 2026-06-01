package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"entsaas/internal/store"

	"github.com/jackc/pgx/v5"
)

type OrganizationExport struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	CreatedAt           time.Time  `json:"created_at"`
	SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
	SuspendedReason     *string    `json:"suspended_reason,omitempty"`
	DeletionScheduledAt *time.Time `json:"deletion_scheduled_at,omitempty"`
}

type UserExport struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectExport struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type InviteExport struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuditLogExport struct {
	ID         string         `json:"id"`
	ActorID    *string        `json:"actor_id,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type ExportPayload struct {
	Organization *OrganizationExport `json:"organization"`
	Users        []*UserExport       `json:"users"`
	Projects     []*ProjectExport    `json:"projects"`
	Invites      []*InviteExport     `json:"invites"`
	AuditLogs    []*AuditLogExport   `json:"audit_logs"`
}

func handleDataCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("data", flag.ExitOnError)
	orgIDOpt := fs.String("org-id", "", "ID of the organization")
	formatOpt := fs.String("format", "json", "Export format (json or csv)")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	if *orgIDOpt == "" {
		fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'data'")
		os.Exit(2)
	}

	switch verb {
	case "export":
		if *formatOpt != "json" && *formatOpt != "csv" {
			fmt.Fprintf(os.Stderr, "Error: Invalid format '%s'. Supported formats are 'json' and 'csv'.\n", *formatOpt)
			os.Exit(2)
		}
		runDataExport(ctx, pgStore, *orgIDOpt, *formatOpt)
	case "purge":
		runDataPurge(ctx, pgStore, *orgIDOpt)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'data'\n", verb)
		os.Exit(2)
	}
}

func runDataExport(ctx context.Context, pgStore *store.PostgresStore, orgID string, format string) {
	// Let's gather the data.
	// 1. Organization
	org, err := pgStore.GetOrganizationByID(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Organization %s not found: %v\n", orgID, err)
		os.Exit(1)
	}

	orgExp := &OrganizationExport{
		ID:                  org.ID,
		Name:                org.Name,
		Slug:                org.Slug,
		CreatedAt:           org.CreatedAt,
		SuspendedAt:         org.SuspendedAt,
		SuspendedReason:     org.SuspendedReason,
		DeletionScheduledAt: org.DeletionScheduledAt,
	}

	// 2. Users
	users, err := pgStore.GetUsersByOrg(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching users: %v\n", err)
		os.Exit(1)
	}
	userExps := make([]*UserExport, 0, len(users))
	for _, u := range users {
		userExps = append(userExps, &UserExport{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
		})
	}

	// 3. Projects
	projects, err := pgStore.ListProjectsByOrg(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching projects: %v\n", err)
		os.Exit(1)
	}
	projExps := make([]*ProjectExport, 0, len(projects))
	for _, p := range projects {
		projExps = append(projExps, &ProjectExport{
			ID:        p.ID,
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
		})
	}

	// 4. Invites
	invites, err := pgStore.GetInvitesByOrg(ctx, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching invites: %v\n", err)
		os.Exit(1)
	}
	invExps := make([]*InviteExport, 0, len(invites))
	for _, i := range invites {
		invExps = append(invExps, &InviteExport{
			ID:        i.ID,
			Email:     i.Email,
			Role:      i.Role,
			CreatedBy: i.CreatedBy,
			CreatedAt: i.CreatedAt,
			ExpiresAt: i.ExpiresAt,
		})
	}

	// 5. Audit logs
	logs, _, err := pgStore.GetAuditLogs(ctx, orgID, 200, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching audit logs: %v\n", err)
		os.Exit(1)
	}
	logExps := make([]*AuditLogExport, 0, len(logs))
	for _, l := range logs {
		logExps = append(logExps, &AuditLogExport{
			ID:         l.ID.String(),
			ActorID:    l.ActorID,
			Action:     l.Action,
			EntityType: l.EntityType,
			EntityID:   l.EntityID,
			Metadata:   l.Metadata,
			CreatedAt:  l.CreatedAt,
		})
	}

	payload := ExportPayload{
		Organization: orgExp,
		Users:        userExps,
		Projects:     projExps,
		Invites:      invExps,
		AuditLogs:    logExps,
	}

	if format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(payload); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Serialization failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Output clean combined CSV layout
		w := csv.NewWriter(os.Stdout)

		// Section: Organization
		_ = w.Write([]string{"--- ORGANIZATION ---"})
		_ = w.Write([]string{"ID", "NAME", "SLUG", "CREATED_AT"})
		_ = w.Write([]string{orgExp.ID, orgExp.Name, orgExp.Slug, orgExp.CreatedAt.Format(time.RFC3339)})
		w.Flush()

		// Section: Users
		fmt.Println()
		_ = w.Write([]string{"--- USERS ---"})
		_ = w.Write([]string{"ID", "EMAIL", "ROLE", "ACTIVE", "CREATED_AT"})
		for _, u := range userExps {
			_ = w.Write([]string{u.ID, u.Email, u.Role, strconv.FormatBool(u.IsActive), u.CreatedAt.Format(time.RFC3339)})
		}
		w.Flush()

		// Section: Projects
		fmt.Println()
		_ = w.Write([]string{"--- PROJECTS ---"})
		_ = w.Write([]string{"ID", "NAME", "CREATED_AT"})
		for _, p := range projExps {
			_ = w.Write([]string{p.ID, p.Name, p.CreatedAt.Format(time.RFC3339)})
		}
		w.Flush()

		// Section: Invites
		fmt.Println()
		_ = w.Write([]string{"--- INVITES ---"})
		_ = w.Write([]string{"ID", "EMAIL", "ROLE", "CREATED_BY", "CREATED_AT"})
		for _, i := range invExps {
			_ = w.Write([]string{i.ID, i.Email, i.Role, i.CreatedBy, i.CreatedAt.Format(time.RFC3339)})
		}
		w.Flush()
	}
}

func runDataPurge(ctx context.Context, pgStore *store.PostgresStore, orgID string) {
	opts := GateOptions{
		Action:      "data.purge",
		Risk:        RiskCritical,
		OrgID:       orgID,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		// Run cascading transactional purge
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
		fmt.Fprintf(os.Stderr, "Error: Data purge failed: %v\n", err)
		os.Exit(1)
	}
}
