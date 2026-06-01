package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"entsaas/internal/store"
)

func handleAuditLogCommand(ctx context.Context, pgStore *store.PostgresStore, args []string) {
	fs := flag.NewFlagSet("audit-log", flag.ExitOnError)
	orgIDOpt := fs.String("org-id", "", "ID of the organization")
	limitOpt := fs.Int("limit", 50, "Maximum number of logs to retrieve (default: 50)")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	if *orgIDOpt == "" {
		fmt.Fprintln(os.Stderr, "Error: --org-id is required for 'audit-log'")
		os.Exit(2)
	}

	runAuditLogList(ctx, pgStore, *orgIDOpt, *limitOpt)
}

func runAuditLogList(ctx context.Context, pgStore *store.PostgresStore, orgID string, limit int) {
	logs, _, err := pgStore.GetAuditLogs(ctx, orgID, limit, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch audit logs: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tACTOR\tACTION\tENTITY TYPE\tENTITY ID")

	for _, entry := range logs {
		actor := "-"
		if entry.ActorID != nil {
			actor = *entry.ActorID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			entry.CreatedAt.Format("2006-01-02 15:04:05"),
			actor,
			entry.Action,
			entry.EntityType,
			entry.EntityID,
		)
	}

	_ = w.Flush()
	if len(logs) == 0 {
		fmt.Println("No audit log entries found.")
	}
}
