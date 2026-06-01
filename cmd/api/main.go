package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"entsaas/internal/ai"
	"entsaas/internal/api"
	"entsaas/internal/billing"
	"entsaas/internal/bootstrap"
	"entsaas/internal/mail"
	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func main() {
	bootstrap.InitLogging()

	// ── Config ──────────────────────────────────────────────────────────────
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal().Msg("DATABASE_URL is required")
	}
	mk := bootstrap.MustParseMasterKey()

	// ── Redis (optional) ─────────────────────────────────────────────────────
	redisURL := os.Getenv("REDIS_URL")
	redisStore, err := store.NewRedisStore(redisURL)
	if err != nil {
		log.Warn().Err(err).Msg("Redis unavailable, continuing without cache")
	}

	// ── Postgres ─────────────────────────────────────────────────────────────
	poolCfg := bootstrap.PoolConfigFromEnv()
	appStore, err := store.NewPostgresStore(dsn, mk.Key, mk.Version, poolCfg.MaxConns, poolCfg.MinConns)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Postgres")
	}
	defer appStore.Close()

	// ── AI Config ───────────────────────────────────────────────────────────
	aiCfg := ai.LoadConfig()

	// ── Mail ────────────────────────────────────────────────────────────────
	mailer := mail.New()

	// ── Billing Catalog ──────────────────────────────────────────────────────
	billingConfigPath := bootstrap.GetEnvStr("ENTSAAS_BILLING_CONFIG", "config/billing.yaml")
	if err := billing.LoadConfig(billingConfigPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to load billing configuration")
	}

	// Sync config plans to database
	ctxSync, cancelSync := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelSync()
	for _, p := range billing.GlobalCatalog.Plans {
		// convert Entitlements struct to map for the generic Upsert signature
		entMap := map[string]any{
			"max_projects":             p.Capacity.MaxProjects,
			"max_members":              p.Capacity.MaxMembers,
			"audit_log_retention_days": p.Capacity.AuditLogRetentionDays,
			"ai_assistant_enabled":     p.Features.AIAssistantEnabled,
			"sso_enabled":              p.Features.SSOEnabled,
			"priority_support":         p.Features.PrioritySupport,
		}
		if err := appStore.SyncBillingPlan(ctxSync, p.ID, p.Name, p.Description, entMap, p.IsSelfServe); err != nil {
			log.Fatal().Err(err).Str("plan", p.ID).Msg("Failed to sync billing plan to database")
		}
	}
	log.Info().Msg("Billing catalog synchronized with database")

	// ── Billing Provider ────────────────────────────────────────────────────
	bi := billing.New()

	// ── Router ──────────────────────────────────────────────────────────────
	r := api.NewRouter(appStore, redisStore, mk.Key, aiCfg, mailer, bi)

	// ── SPA (Static Files) ──────────────────────────────────────────────────
	workDir, _ := os.Getwd()
	webDist := workDir + "/web/dist"

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		fullPath := webDist + path

		if _, err := os.Stat(fullPath); err == nil && path != "/" {
			if strings.HasPrefix(path, "/assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
			c.File(fullPath)
			return
		}

		if strings.HasPrefix(path, "/assets/") {
			c.Status(http.StatusNotFound)
			return
		}

		if os.Getenv("ENTSAAS_DEV") == "1" {
			c.Header("Cache-Control", "no-store, no-cache")
		}
		c.File(webDist + "/index.html")
	})

	// ── HTTP Server ─────────────────────────────────────────────────────────
	port := bootstrap.GetEnvStr("PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Msgf("EntSaaS API server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("API server startup failed")
		}
	}()

	// ── Graceful shutdown ───────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("API server forced to shutdown")
	}
	log.Info().Msg("EntSaaS API server exiting")
}
