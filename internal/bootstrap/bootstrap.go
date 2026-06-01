// Package bootstrap provides shared config/env helpers used by multiple
// cmd/* binaries. This is NOT a config framework — just deduplicated
// helpers that are stable, repeated, and unlikely to vary.
package bootstrap

import (
	"encoding/hex"
	"os"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogging sets up zerolog with console output and Unix timestamps.
func InitLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

// MasterKeyConfig holds the parsed master key and version.
type MasterKeyConfig struct {
	Key     []byte
	Version int
}

// MustParseMasterKey reads ENTSAAS_MASTER_KEY from env, hex-decodes it,
// and returns the key bytes + version. Fatals on invalid hex.
func MustParseMasterKey() MasterKeyConfig {
	raw := os.Getenv("ENTSAAS_MASTER_KEY")
	key, err := hex.DecodeString(raw)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid master key hex (ENTSAAS_MASTER_KEY)")
	}
	version := GetEnvInt("ENTSAAS_MASTER_KEY_VERSION", 1)
	return MasterKeyConfig{Key: key, Version: version}
}

// GetEnvStr reads an env var as string, returning def if unset.
func GetEnvStr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// GetEnvInt reads an env var as int, returning def on missing or unparseable.
func GetEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GetEnvInt64 reads an env var as int64, returning def on missing or unparseable.
func GetEnvInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// GetEnvBool reads an env var as boolean. Returns def on missing or unparseable.
// Accepts: "true", "1", "yes" (case-insensitive).
func GetEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// PoolConfig holds database connection pool sizing.
type PoolConfig struct {
	MaxConns int32
	MinConns int32
}

// PoolConfigFromEnv reads connection pool sizing from env vars.
// Zero values mean "use the package default" (MaxConns=25, MinConns=5).
func PoolConfigFromEnv() PoolConfig {
	return PoolConfig{
		MaxConns: int32(GetEnvInt("DB_MAX_CONNS", 0)),
		MinConns: int32(GetEnvInt("DB_MIN_CONNS", 0)),
	}
}
