package main

import (
	"database/sql"
	"flag"
	"os"

	"entsaas/internal/bootstrap"
	"entsaas/internal/migrations/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func main() {
	bootstrap.InitLogging()

	command := flag.String("command", "up", "Migration command: up, down, status, reset")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal().Msg("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open DB for migrations")
	}
	defer db.Close()

	goose.SetBaseFS(postgres.MigrationsFS)

	switch *command {
	case "up":
		if err := goose.Up(db, "."); err != nil {
			log.Fatal().Err(err).Msg("Migration up failed")
		}
		log.Info().Msg("Migrations applied successfully")
	case "down":
		if err := goose.Down(db, "."); err != nil {
			log.Fatal().Err(err).Msg("Migration down failed")
		}
		log.Info().Msg("Migration rolled back")
	case "status":
		if err := goose.Status(db, "."); err != nil {
			log.Fatal().Err(err).Msg("Migration status failed")
		}
	case "reset":
		if err := goose.Reset(db, "."); err != nil {
			log.Fatal().Err(err).Msg("Migration reset failed")
		}
		log.Info().Msg("All migrations reset")
	default:
		log.Fatal().Str("command", *command).Msg("Unknown migration command")
	}
}
