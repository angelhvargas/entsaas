package store

import (
	"context"
	"os"
	"testing"

	"entsaas/internal/bootstrap"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestStoreForRouter(t *testing.T) *PostgresStore {
	if testing.Short() {
		t.Skip("skipping DB integration test in short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"
	}
	masterKeyConf := bootstrap.MustParseMasterKey()
	pgStore, err := NewPostgresStore(dsn, masterKeyConf.Key, masterKeyConf.Version, 2, 1)
	if err != nil {
		t.Skipf("skipping DB integration test — no Postgres available: %v", err)
	}
	return pgStore
}

func TestDynamicTenantStoreResolver_FallbackToShared(t *testing.T) {
	systemStore := getTestStoreForRouter(t)
	defer systemStore.Close()

	masterKeyConf := bootstrap.MustParseMasterKey()
	resolver := NewDynamicTenantStoreResolver(systemStore, masterKeyConf.Key, masterKeyConf.Version)
	defer resolver.Close()

	ctx := context.Background()
	tenantID := uuid.New().String()

	// Resolving a new non-existing tenant ID should gracefully log and fallback to systemStore!
	store, err := resolver.ResolveStore(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to resolve tenant store: %v", err)
	}

	if store != systemStore {
		t.Errorf("expected resolved store to be systemStore on fallback, got different store")
	}
}

func TestDynamicTenantStoreResolver_DedicatedPoolResolution(t *testing.T) {
	systemStore := getTestStoreForRouter(t)
	defer systemStore.Close()

	masterKeyConf := bootstrap.MustParseMasterKey()
	resolver := NewDynamicTenantStoreResolver(systemStore, masterKeyConf.Key, masterKeyConf.Version)
	defer resolver.Close()

	// Pre-register a dedicated pool using system database DSN (as a test stand-in)
	tenantID := "premium-customer-123"
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"
	}

	err := resolver.RegisterTenantDSN(tenantID, dsn)
	if err != nil {
		t.Fatalf("failed to pre-register DSN: %v", err)
	}

	ctx := context.Background()
	resolved, err := resolver.ResolveStore(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to resolve registered tenant: %v", err)
	}

	if resolved == systemStore {
		t.Error("expected to resolve dedicated tenant pool, got shared systemStore")
	}

	// Verify it was registered in cache
	resolver.mu.RLock()
	cachedStore, cached := resolver.tenantPools[tenantID]
	resolver.mu.RUnlock()

	if !cached || cachedStore != resolved {
		t.Error("resolved store was not cached in tenant registry")
	}

	// Verify that a query on the dedicated pool works!
	var one int
	pgProvider, ok := resolved.(interface { Pool() *pgxpool.Pool })
	if !ok {
		t.Fatal("resolved store does not support pool provider")
	}
	err = pgProvider.Pool().QueryRow(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		t.Fatalf("failed to query select 1 on resolved tenant pool: %v", err)
	}
	if one != 1 {
		t.Errorf("got %d, want 1", one)
	}

	// Verify lookup in DB mapping fails gracefully if query errors (table doesn't exist yet)
	// Querying will try database lookup first, which fails on scan/sql error, falling back to systemStore.
	failID := "non-exist-" + uuid.New().String()
	fallbackStore, err := resolver.ResolveStore(ctx, failID)
	if err != nil {
		t.Fatalf("ResolveStore returned error, want fallback: %v", err)
	}
	if fallbackStore != systemStore {
		t.Error("expected fallback to shared systemStore")
	}
}

func TestDynamicTenantStoreResolver_MetadataQueryFallback(t *testing.T) {
	systemStore := getTestStoreForRouter(t)
	defer systemStore.Close()

	masterKeyConf := bootstrap.MustParseMasterKey()
	resolver := NewDynamicTenantStoreResolver(systemStore, masterKeyConf.Key, masterKeyConf.Version)
	defer resolver.Close()

	ctx := context.Background()

	// Resolving empty tenant ID should immediately return systemStore without queries
	store, err := resolver.ResolveStore(ctx, "")
	if err != nil {
		t.Fatalf("failed to resolve empty tenant ID: %v", err)
	}
	if store != systemStore {
		t.Error("expected empty tenant ID to map to systemStore")
	}
}

func TestDynamicTenantStoreResolver_RegistryIsolationClose(t *testing.T) {
	systemStore := getTestStoreForRouter(t)
	defer systemStore.Close()

	masterKeyConf := bootstrap.MustParseMasterKey()
	resolver := NewDynamicTenantStoreResolver(systemStore, masterKeyConf.Key, masterKeyConf.Version)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"
	}

	_ = resolver.RegisterTenantDSN("t1", dsn)
	resolver.Close()

	// Registry must be cleared
	if len(resolver.tenantPools) != 0 {
		t.Errorf("tenant pools registry was not cleared on Close, size: %d", len(resolver.tenantPools))
	}

	// Dynamic resolve on closed resolver must fallback or safely recreate
	fallback, err := resolver.ResolveStore(context.Background(), "")
	if err != nil {
		t.Fatalf("failed to resolve empty tenant after Close: %v", err)
	}
	if fallback != systemStore {
		t.Error("expected fallback to systemStore")
	}
}
