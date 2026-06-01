package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"entsaas/internal/bootstrap"
	"entsaas/internal/store"
)

var (
	dryRunFlag  bool
	yesFlag     bool
	timeoutFlag string
	reasonFlag  string
)

func initGlobalFlags(fs *flag.FlagSet) {
	fs.BoolVar(&dryRunFlag, "dry-run", false, "Preview the operation without modifying data")
	fs.BoolVar(&yesFlag, "yes", false, "Bypass interactive prompts (requires ENTSAASCTL_ALLOW_UNATTENDED=true)")
	fs.StringVar(&timeoutFlag, "timeout", "", "Timeout duration (e.g. 10s, 5m)")
	fs.StringVar(&reasonFlag, "reason", "", "Mandatory explanation for High/Critical risk operations")
}

func printUsage() {
	fmt.Println(`entsaasctl — EntSaaS Secure Operations Command-Line Tool

Usage:
  entsaasctl <resource> <verb> [flags]

Safe Operations (Read-Only):
  entsaasctl ping
  entsaasctl orgs list
  entsaasctl orgs inspect --org-id <id>
  entsaasctl users list --org-id <id>
  entsaasctl projects list --org-id <id>
  entsaasctl audit-log --org-id <id> [--limit N]
  entsaasctl tokens list-active --user-id <id>

Mutating Operations (Dangerous — Safety Gates Apply):
  entsaasctl users set-role --user-id <id> --org-id <id> --role <role>
  entsaasctl users set-status --user-id <id> --org-id <id> --active <bool>
  entsaasctl users revoke-sessions --user-id <id>
  entsaasctl orgs suspend --org-id <id> --reason <text>
  entsaasctl orgs unsuspend --org-id <id>
  entsaasctl orgs delete --org-id <id> [--immediate]
  entsaasctl secrets rotate --scope <jwt|master-key> [--grace <duration>]
  entsaasctl data export --org-id <id> [--format json|csv]
  entsaasctl data purge --org-id <id>

Global Flags:
  --dry-run      Preview the operation without modifying data
  --yes          Bypass interactive prompts (requires ENTSAASCTL_ALLOW_UNATTENDED=true)
  --timeout      Configure execution timeout (e.g., 30s)
  --reason       Provide justification for High/Critical mutations

Exit Codes:
  0: Success
  1: Unspecified runtime error
  2: Command-line syntax error
  3: Safety gate rejected (confirmation declined, dry-run completed)
  4: Timeout or connection failure
  5: Integrity error (rolled back)`)
}

func auditAndRedactDSN(dsn string) string {
	if strings.Contains(dsn, "://") {
		parts := strings.SplitN(dsn, "@", 2)
		if len(parts) == 2 {
			left := parts[0]
			right := parts[1]
			subParts := strings.SplitN(left, "://", 2)
			if len(subParts) == 2 {
				prefix := subParts[0]
				creds := subParts[1]
				credParts := strings.SplitN(creds, ":", 2)
				if len(credParts) == 2 && credParts[1] != "" {
					fmt.Fprintln(os.Stderr, "⚠️  DSN contains plaintext password. Use pg_service.conf or .pgpass in production.")
					return fmt.Sprintf("%s://%s:****@%s", prefix, credParts[0], right)
				}
			}
		}
	}
	return dsn
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	resource := os.Args[1]

	// Handle standard help commands
	if resource == "help" || resource == "-h" || resource == "--help" {
		printUsage()
		os.Exit(0)
	}

	// 1. DSN setup and auditing
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Attempt local default
		dsn = "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"
	}
	_ = auditAndRedactDSN(dsn)

	// 2. Parse Master Key from environment
	masterKeyConf := bootstrap.MustParseMasterKey()

	// 3. Initialize PostgresStore with connection limits (MaxConns=2, MinConns=1)
	pgStore, err := store.NewPostgresStore(dsn, masterKeyConf.Key, masterKeyConf.Version, 2, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to database: %v\n", err)
		os.Exit(4)
	}
	defer pgStore.Close()

	ctx := context.Background()

	// Dispatch commands
	switch resource {
	case "ping":
		runPing(ctx, pgStore)

	case "orgs":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'orgs' (list, inspect, suspend, unsuspend, delete)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleOrgsCommand(ctx, pgStore, verb, os.Args[3:])

	case "users":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'users' (list, set-role, set-status, revoke-sessions)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleUsersCommand(ctx, pgStore, verb, os.Args[3:])

	case "projects":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'projects' (list)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleProjectsCommand(ctx, pgStore, verb, os.Args[3:])

	case "audit-log":
		handleAuditLogCommand(ctx, pgStore, os.Args[2:])

	case "tokens":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'tokens' (list-active)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleTokensCommand(ctx, pgStore, verb, os.Args[3:])

	case "secrets":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'secrets' (rotate)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleSecretsCommand(ctx, pgStore, verb, os.Args[3:])

	case "data":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: Missing sub-action for 'data' (export, purge)")
			os.Exit(2)
		}
		verb := os.Args[2]
		handleDataCommand(ctx, pgStore, verb, os.Args[3:])

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown resource '%s'\n", resource)
		printUsage()
		os.Exit(2)
	}
}

func runPing(ctx context.Context, pgStore *store.PostgresStore) {
	err := pgStore.Pool().Ping(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "🔴 Connection Failed: %v\n", err)
		os.Exit(4)
	}
	fmt.Println("💚 Database connection successful.")
}
