package api

import (
	"entsaas/internal/ai"
	"entsaas/internal/billing"
	"entsaas/internal/mail"
	"entsaas/internal/middleware"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// NewRouter creates and returns the Gin engine for the API server.
//
// This is the composition layer: it creates the engine, applies base
// middleware, builds shared dependencies, and delegates route registration
// to per-surface registration functions.
//
// Surfaces:
//   - PUBLIC  — unauthenticated (health, auth endpoints)
//   - APP     — session-authenticated product UI
//   - ADMIN   — elevated authorization / org-admin
func NewRouter(
	appStore store.AppStore,
	redisStore *store.RedisStore,
	masterKey []byte,
	aiCfg ai.Config,
	mailer mail.Sender,
	bi billing.Provider,
) *gin.Engine {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Body-size limit: prevent memory exhaustion from oversized payloads.
	r.MaxMultipartMemory = 1 << 20 // 1 MiB

	// ── Base Middleware ──────────────────────────────────────────────────────
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.MaxBodySize(1 << 20)) // SEC-14: 1 MiB JSON body limit
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())

	// Extract redis.Client for rate limiting middleware (may be nil).
	var redisClient *redis.Client
	if redisStore != nil {
		redisClient = redisStore.Client
	}
	_ = redisClient // available for future rate limiting middleware

	deps := RouterDeps{
		Store:       appStore,
		RedisStore:  redisStore,
		MasterKey:   masterKey,
		RedisClient: redisClient,
		AIConfig:    aiCfg,
		Mailer:      mailer,
		Billing:     bi,
	}

	// ── API v1 ──────────────────────────────────────────────────────────────
	v1 := r.Group("/v1")

	// ── Surface Registration ────────────────────────────────────────────────
	RegisterPublicRoutes(r, v1, deps)
	RegisterAppRoutes(v1, deps)
	RegisterAdminRoutes(v1, deps)

	return r
}
