package main

import (
	"context"
	"os"
	"time"

	"entsaas/internal/bootstrap"
	"entsaas/internal/store"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	bootstrap.InitLogging()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal().Msg("DATABASE_URL is required")
	}

	email := bootstrap.GetEnvStr("ENTSAAS_ADMIN_EMAIL", "admin@entsaas.dev")
	password := bootstrap.GetEnvStr("ENTSAAS_ADMIN_PASSWORD", "password")

	mk := bootstrap.MustParseMasterKey()
	poolCfg := bootstrap.PoolConfigFromEnv()

	appStore, err := store.NewPostgresStore(dsn, mk.Key, mk.Version, poolCfg.MaxConns, poolCfg.MinConns)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Postgres")
	}
	defer appStore.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create default org.
	org, err := appStore.CreateOrganization(ctx, "Default", "default")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create org")
	}
	log.Info().Str("org_id", org.ID).Str("slug", org.Slug).Msg("Organization ready")

	// Create admin user.
	user, err := appStore.CreateUser(ctx, email, "owner", org.ID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create admin user")
	}
	log.Info().Str("user_id", user.ID).Str("email", email).Msg("Admin user created")

	// Set password.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to hash password")
	}
	if err := appStore.CreateUserCredential(ctx, user.ID, string(hash)); err != nil {
		log.Fatal().Err(err).Msg("Failed to store credentials")
	}

	// Mark email as verified.
	_ = appStore.MarkUserEmailVerified(ctx, user.ID)

	log.Info().
		Str("email", email).
		Str("password", password).
		Msg("✅ Seed complete. Login with these credentials.")
}
