-- PostgreSQL init script for local development.
-- Executed automatically by the postgres container on first start.

-- Enable useful extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
