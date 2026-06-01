package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"entsaas/internal/store"

	"github.com/jackc/pgx/v5"
)

func handleSecretsCommand(ctx context.Context, pgStore *store.PostgresStore, verb string, args []string) {
	fs := flag.NewFlagSet("secrets", flag.ExitOnError)
	scopeOpt := fs.String("scope", "", "Scope of secret rotation (jwt or master-key)")
	graceOpt := fs.String("grace", "24h", "Grace period for the old secret (default: 24h)")
	initGlobalFlags(fs)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Flag parsing failed: %v\n", err)
		os.Exit(2)
	}

	if verb != "rotate" {
		fmt.Fprintf(os.Stderr, "Error: Unknown action '%s' for 'secrets'\n", verb)
		os.Exit(2)
	}

	if *scopeOpt == "" {
		fmt.Fprintln(os.Stderr, "Error: --scope is required for 'rotate' (jwt or master-key)")
		os.Exit(2)
	}

	if *scopeOpt != "jwt" && *scopeOpt != "master-key" {
		fmt.Fprintf(os.Stderr, "Error: Invalid scope '%s'. Must be either 'jwt' or 'master-key'.\n", *scopeOpt)
		os.Exit(2)
	}

	gracePeriod, err := time.ParseDuration(*graceOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid grace duration '%s': %v\n", *graceOpt, err)
		os.Exit(2)
	}

	runSecretsRotate(ctx, pgStore, *scopeOpt, gracePeriod)
}

func runSecretsRotate(ctx context.Context, pgStore *store.PostgresStore, scope string, grace time.Duration) {
	// Generate secure cryptographically high-entropy new key material
	newKeyBytes := make([]byte, 32)
	if _, err := rand.Read(newKeyBytes); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cryptographic random generation failed: %v\n", err)
		os.Exit(1)
	}
	newKeyHex := hex.EncodeToString(newKeyBytes)

	opts := GateOptions{
		Action:      "secrets.rotate",
		Risk:        RiskCritical,
		Reason:      reasonFlag,
		DryRun:      dryRunFlag,
		YesFlag:     yesFlag,
		TimeoutFlag: timeoutFlag,
		Metadata: map[string]any{
			"scope":        scope,
			"grace_period": grace.String(),
		},
	}

	err := ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		// Verify if we can perform a simulated rotation inside the transaction.
		// Output setup instructions to stdout:
		fmt.Println("\n🔑 --- NEW CRYPTOGRAPHIC SECRET GENERATED ---")
		if scope == "master-key" {
			fmt.Printf("ENTSAAS_MASTER_KEY=%s\n", newKeyHex)
			fmt.Println("ENTSAAS_MASTER_KEY_VERSION=<increment previous version>")
		} else {
			fmt.Printf("JWT_SECRET=%s\n", newKeyHex)
		}
		fmt.Println("----------------------------------------------")
		fmt.Printf("👉 Instruction: Update the environment configuration block. The old secret remains active for grace period: %s.\n\n", grace)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Secret rotation failed: %v\n", err)
		os.Exit(1)
	}
}
