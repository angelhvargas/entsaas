package api

import (
	"entsaas/internal/ai"
	"entsaas/internal/billing"
	"entsaas/internal/mail"
	"entsaas/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// RouterDeps holds all shared dependencies for route registration.
// This struct is the composition root — handlers receive only what they need.
type RouterDeps struct {
	Store       store.AppStore
	RedisStore  *store.RedisStore
	Pool        *pgxpool.Pool
	RedisClient *redis.Client
	MasterKey   []byte
	AIConfig    ai.Config
	Mailer      mail.Sender    // nil-safe: LogSender used when not configured
	Billing     billing.Provider // nil-safe: NoopProvider used when not configured
}
