package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"entsaas/internal/store"
)

func handleTokensCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("tokens", flag.ExitOnError)
	userIDOpt := fs.String("user-id", "", "ID of the user")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	if verb != "list-active" {
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'tokens'\n", verb)
		os.Exit(2)
	}

	if *userIDOpt == "" {
		fmt.Fprintln(os.Stderr, "Error: --user-id is required for 'list-active'")
		os.Exit(2)
	}

	runTokensListActive(ctx, pgStore, *userIDOpt)
}

func runTokensListActive(ctx context.Context, pgStore *store.PostgresStore, userID string) {
	rows, err := pgStore.Pool().Query(ctx,
		`SELECT token_hash, org_id, user_agent, ip_address, expires_at, created_at
		 FROM refresh_tokens
		 WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to fetch active tokens: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOKEN HASH PREFIX\tORG ID\tUSER AGENT\tIP ADDRESS\tCREATED AT\tEXPIRES AT")

	count := 0
	for rows.Next() {
		var tokenHash, orgID, userAgent, ipAddress string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&tokenHash, &orgID, &userAgent, &ipAddress, &expiresAt, &createdAt); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning tokens: %v\n", err)
			os.Exit(1)
		}

		// Show only a safe prefix of the token hash
		hashPrefix := tokenHash
		if len(hashPrefix) > 10 {
			hashPrefix = hashPrefix[:10] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			hashPrefix,
			orgID,
			userAgent,
			ipAddress,
			createdAt.Format("2006-01-02 15:04:05"),
			expiresAt.Format("2006-01-02 15:04:05"),
		)
		count++
	}

	_ = w.Flush()
	if count == 0 {
		fmt.Println("No active refresh tokens found for this user.")
	}
}
