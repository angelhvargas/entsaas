package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var ErrTenantNotFound = errors.New("tenant: database pool resolution failed")

// TenantStoreResolver resolves isolated AppStore handles dynamically based on tenant ID.
type TenantStoreResolver interface {
	ResolveStore(ctx context.Context, tenantID string) (AppStore, error)
	Close()
}

// DynamicTenantStoreResolver manages dedicated connection pools per customer.
type DynamicTenantStoreResolver struct {
	mu           sync.RWMutex
	systemStore  AppStore
	masterKey    []byte
	masterKeyVer int
	tenantPools  map[string]AppStore
}

// NewDynamicTenantStoreResolver creates a new DynamicTenantStoreResolver.
func NewDynamicTenantStoreResolver(systemStore AppStore, masterKey []byte, masterKeyVer int) *DynamicTenantStoreResolver {
	return &DynamicTenantStoreResolver{
		systemStore:  systemStore,
		masterKey:    masterKey,
		masterKeyVer: masterKeyVer,
		tenantPools:  make(map[string]AppStore),
	}
}

// ResolveStore fetches or dynamically provisions a dedicated database connection pool for the customer.
// Fallback: If no dedicated DSN is found in the metadata lookup, routes to the shared System pool.
func (r *DynamicTenantStoreResolver) ResolveStore(ctx context.Context, tenantID string) (AppStore, error) {
	if tenantID == "" {
		return r.systemStore, nil
	}

	// 1. Read lock check
	r.mu.RLock()
	store, exists := r.tenantPools[tenantID]
	r.mu.RUnlock()

	if exists {
		return store, nil
	}

	// 2. Write lock check and dynamic provisioning
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check under write lock
	if store, exists = r.tenantPools[tenantID]; exists {
		return store, nil
	}

	// Dynamic Metadata Lookup: query system metadata for tenant DSN
	var dsn string
	// If a dedicated tenant mapping table exists (e.g. tenant_databases), query it:
	// We simulate this by checking a system query.
	query := `SELECT db_dsn FROM tenant_databases WHERE tenant_id = $1`
	
	var err error
	if pgProvider, ok := r.systemStore.(interface { Pool() *pgxpool.Pool }); ok {
		err = pgProvider.Pool().QueryRow(ctx, query, tenantID).Scan(&dsn)
	} else {
		err = errors.New("system store does not support pool queries")
	}

	if err != nil {
		// Log and gracefully fall back to the shared pool (logical isolation)
		log.Debug().Str("tenant_id", tenantID).Msg("No dedicated DSN resolved, falling back to shared system pool")
		r.tenantPools[tenantID] = r.systemStore
		return r.systemStore, nil
	}

	// provision a dedicated connection pool
	log.Info().Str("tenant_id", tenantID).Msg("Provisioning dedicated tenant database connection pool...")
	tenantStore, err := NewPostgresStore(dsn, r.masterKey, r.masterKeyVer, 10, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to provision dedicated pool for tenant %q: %w", tenantID, err)
	}

	r.tenantPools[tenantID] = tenantStore
	log.Info().Str("tenant_id", tenantID).Msg("Dedicated tenant database pool active")
	return tenantStore, nil
}

// RegisterTenantDSN manual utility to pre-register customer database pools (e.g. for bootstrapping/tests)
func (r *DynamicTenantStoreResolver) RegisterTenantDSN(tenantID, dsn string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tenantStore, err := NewPostgresStore(dsn, r.masterKey, r.masterKeyVer, 5, 1)
	if err != nil {
		return err
	}
	r.tenantPools[tenantID] = tenantStore
	return nil
}

// Close gracefully closes all active dynamically allocated tenant database connection pools.
func (r *DynamicTenantStoreResolver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tenantID, store := range r.tenantPools {
		// Do not close the shared system store (system main handles it)
		if store != r.systemStore {
			log.Info().Str("tenant_id", tenantID).Msg("Closing dedicated database pool...")
			store.Close()
		}
	}
	r.tenantPools = nil
}
