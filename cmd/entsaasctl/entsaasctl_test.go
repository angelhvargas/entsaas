package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"entsaas/internal/bootstrap"
	"entsaas/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestMain(m *testing.M) {
	// Set env vars required for tests
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable")
	}
	if os.Getenv("ENTSAAS_MASTER_KEY") == "" {
		os.Setenv("ENTSAAS_MASTER_KEY", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	}
	os.Exit(m.Run())
}

func getTestStore(t *testing.T) *store.PostgresStore {
	if testing.Short() {
		t.Skip("skipping DB integration test in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	masterKeyConf := bootstrap.MustParseMasterKey()
	pgStore, err := store.NewPostgresStore(dsn, masterKeyConf.Key, masterKeyConf.Version, 2, 1)
	if err != nil {
		t.Skipf("skipping DB integration test — no Postgres available: %v", err)
	}
	return pgStore
}

func TestSafetyGates_DryRun(t *testing.T) {
	pgStore := getTestStore(t)
	defer pgStore.Close()

	ctx := context.Background()

	// 1. Create a dummy organization to test on
	org, err := pgStore.CreateOrganization(ctx, "Test Dry Run Org", "test-dry-run-org-"+uuid.New().String()[:6])
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	// Since opts.DryRun is true, ExecuteWithGate will output and call os.Exit(3)
	// We want to make sure it validates correctly BEFORE executing. Let's verify that a High-risk dry-run without reason fails the validation gate.
	optsNoReason := GateOptions{
		Action:  "orgs.suspend",
		Risk:    RiskHigh,
		OrgID:   org.ID,
		Reason:  "",
		DryRun:  true,
		YesFlag: false,
	}

	err = ExecuteWithGate(ctx, pgStore, optsNoReason, func(ctx context.Context, tx pgx.Tx) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "Reason (--reason) is mandatory") {
		t.Errorf("Expected dry run to fail input validation due to missing reason, got: %v", err)
	}
}

func TestSafetyGates_NonInteractiveBypassRejection(t *testing.T) {
	pgStore := getTestStore(t)
	defer pgStore.Close()

	ctx := context.Background()
	org, err := pgStore.CreateOrganization(ctx, "Test Bypass Org", "test-bypass-org-"+uuid.New().String()[:6])
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	// High risk operation with --yes set but without ENTSAASCTL_ALLOW_UNATTENDED=true
	opts := GateOptions{
		Action:  "orgs.suspend",
		Risk:    RiskHigh,
		OrgID:   org.ID,
		Reason:  "Testing bypass rejection",
		DryRun:  false,
		YesFlag: true,
	}

	os.Setenv("ENTSAASCTL_ALLOW_UNATTENDED", "false")

	err = ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		return nil
	})

	if err == nil || !strings.Contains(err.Error(), "ENTSAASCTL_ALLOW_UNATTENDED=true is NOT set") {
		t.Errorf("Expected safety gate to reject unattended bypass, got: %v", err)
	}
}

func TestSafetyGates_SuccessWithAuditLog(t *testing.T) {
	pgStore := getTestStore(t)
	defer pgStore.Close()

	ctx := context.Background()
	org, err := pgStore.CreateOrganization(ctx, "Test Audit Success", "test-audit-success-"+uuid.New().String()[:6])
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	// Set env vars to allow unattended execution
	os.Setenv("ENTSAASCTL_ALLOW_UNATTENDED", "true")
	defer os.Unsetenv("ENTSAASCTL_ALLOW_UNATTENDED")

	opts := GateOptions{
		Action:  "orgs.suspend",
		Risk:    RiskHigh,
		OrgID:   org.ID,
		Reason:  "Testing successful audit logging",
		DryRun:  false,
		YesFlag: true,
	}

	err = ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE organizations SET suspended_at = NOW(), suspended_reason = $2 WHERE id = $1`,
			org.ID, opts.Reason)
		return err
	})

	if err != nil {
		t.Fatalf("expected successful execution, got: %v", err)
	}

	// Verify that the organization was actually suspended
	dbOrg, err := pgStore.GetOrganizationByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated org: %v", err)
	}
	if dbOrg.SuspendedAt == nil || dbOrg.SuspendedReason == nil || *dbOrg.SuspendedReason != opts.Reason {
		t.Errorf("organization was not suspended properly: %+v", dbOrg)
	}

	// Verify the audit log entry is written transactionally
	var actorID, action, metadataStr string
	err = pgStore.Pool().QueryRow(ctx,
		`SELECT actor_id, action, metadata::text FROM audit_log WHERE org_id = $1 AND action = $2`,
		org.ID, "entsaasctl.orgs.suspend").Scan(&actorID, &action, &metadataStr)

	if err != nil {
		t.Fatalf("failed to find audit log entry: %v", err)
	}

	expectedActor := GetActorString()
	if actorID != expectedActor {
		t.Errorf("expected actor %q, got %q", expectedActor, actorID)
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataStr), &meta); err != nil {
		t.Fatalf("failed to parse metadata json: %v", err)
	}

	if meta["unattended"] != true {
		t.Errorf("expected unattended to be true in audit log metadata, got: %v", meta["unattended"])
	}
	if meta["reason"] != opts.Reason {
		t.Errorf("expected reason %q, got %q", opts.Reason, meta["reason"])
	}
	if meta["success"] != true {
		t.Errorf("expected success to be true in audit log metadata, got: %v", meta["success"])
	}
}

func TestTransactionalAuditLogRollback(t *testing.T) {
	pgStore := getTestStore(t)
	defer pgStore.Close()

	ctx := context.Background()
	org, err := pgStore.CreateOrganization(ctx, "Test Rollback", "test-rollback-"+uuid.New().String()[:6])
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	// Set env vars to allow unattended execution
	os.Setenv("ENTSAASCTL_ALLOW_UNATTENDED", "true")
	defer os.Unsetenv("ENTSAASCTL_ALLOW_UNATTENDED")

	opts := GateOptions{
		Action:  "orgs.suspend",
		Risk:    RiskHigh,
		OrgID:   org.ID,
		Reason:  "Testing transactional rollback",
		DryRun:  false,
		YesFlag: true,
	}

	// Execute a failing mutation
	err = ExecuteWithGate(ctx, pgStore, opts, func(ctx context.Context, tx pgx.Tx) error {
		// Do some change
		_, _ = tx.Exec(ctx, `UPDATE organizations SET suspended_at = NOW() WHERE id = $1`, org.ID)
		// Force a database constraint error or custom error
		return errors.New("forced mutation error")
	})

	if err == nil {
		t.Fatal("expected ExecuteWithGate to return mutation error, got nil")
	}

	// Verify that the organization was NOT suspended (the transaction rolled back!)
	dbOrg, err := pgStore.GetOrganizationByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to fetch org: %v", err)
	}
	if dbOrg.SuspendedAt != nil {
		t.Error("expected organization suspension to be rolled back, but SuspendedAt is non-nil")
	}

	// But verify the audit log was committed with success = false
	var successBool bool
	var errorMsg string
	err = pgStore.Pool().QueryRow(ctx,
		`SELECT (metadata->>'success')::boolean, metadata->>'error' 
		 FROM audit_log WHERE org_id = $1 AND action = $2`,
		org.ID, "entsaasctl.orgs.suspend").Scan(&successBool, &errorMsg)

	if err != nil {
		t.Fatalf("failed to find audit log entry for failed mutation: %v", err)
	}

	if successBool != false {
		t.Errorf("expected success=false in audit log for failed mutation, got true")
	}
	if !strings.Contains(errorMsg, "forced mutation error") {
		t.Errorf("expected error message to contain 'forced mutation error', got %q", errorMsg)
	}
}

func TestPurgeOrganizationData(t *testing.T) {
	pgStore := getTestStore(t)
	defer pgStore.Close()

	ctx := context.Background()

	// 1. Create org
	org, err := pgStore.CreateOrganization(ctx, "Test Purge Org", "test-purge-org-"+uuid.New().String()[:6])
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	// 2. Create users
	user1, err := pgStore.CreateUser(ctx, "purge1@entsaas.dev", "member", org.ID)
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}
	user2, err := pgStore.CreateUser(ctx, "purge2@entsaas.dev", "admin", org.ID)
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	// 3. Create projects
	proj, err := pgStore.CreateProject(ctx, "Purge Project", org.ID)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// 4. Create active sessions (refresh tokens)
	err = pgStore.CreateRefreshToken(ctx, "hash-"+uuid.New().String(), user1.ID, org.ID, "Mozilla", "127.0.0.1", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// 5. Create invites
	inviteHash := "invitehash-" + uuid.New().String()
	_, err = pgStore.CreateInvite(ctx, org.ID, "invitee@entsaas.dev", "member", inviteHash, time.Now().Add(24*time.Hour), user2.ID)
	if err != nil {
		t.Fatalf("failed to create invite: %v", err)
	}

	// 6. Create user preferences
	err = pgStore.SetUserPreferences(ctx, user1.ID, map[string]any{"theme": "dark"})
	if err != nil {
		t.Fatalf("failed to set user preference: %v", err)
	}

	// 7. Perform the transactional purge via store layer method
	err = pgStore.PurgeOrganizationData(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to purge organization data: %v", err)
	}

	// 8. Verify all references are completely deleted
	// Organization
	_, err = pgStore.GetOrganizationByID(ctx, org.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected org to be deleted, got: %v", err)
	}

	// Users
	_, err = pgStore.GetUserByID(ctx, user1.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected user1 to be deleted, got: %v", err)
	}
	_, err = pgStore.GetUserByID(ctx, user2.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected user2 to be deleted, got: %v", err)
	}

	// Projects
	_, err = pgStore.GetProjectByID(ctx, proj.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected project to be deleted, got: %v", err)
	}

	// Invites
	_, err = pgStore.GetInviteByTokenHash(ctx, inviteHash)
	if err != store.ErrNotFound {
		t.Errorf("expected invite to be deleted, got: %v", err)
	}

	// Refresh tokens
	var exists bool
	_ = pgStore.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE org_id = $1)`, org.ID).Scan(&exists)
	if exists {
		t.Error("expected refresh tokens to be deleted")
	}

	// Audit logs
	_ = pgStore.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_log WHERE org_id = $1)`, org.ID).Scan(&exists)
	if exists {
		t.Error("expected audit logs to be deleted")
	}
}
