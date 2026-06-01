.PHONY: all build build-backend build-web clean deps test test-web \
       dev dev-up dev-down down dev-logs migrate seed-dev dev-reseed \
       lint check-local doctor

# ─── Variables ───────────────────────────────────────────────────────────────
API_NAME   = api
MIGRATE    = migrate
SEED       = seed
CLI_NAME   = entsaasctl
WEB_DIR    = web

# ─── Build ───────────────────────────────────────────────────────────────────
all: build

build-backend:
	@echo "Building Go backend..."
	go build -buildvcs=false -o $(API_NAME) ./cmd/api
	go build -buildvcs=false -o $(MIGRATE) ./cmd/migrate
	go build -buildvcs=false -o $(SEED) ./cmd/seed
	go build -buildvcs=false -o $(CLI_NAME) ./cmd/entsaasctl

build-web:
	@echo "Building Vue dashboard..."
	cd $(WEB_DIR) && npm run build

build: build-backend build-web

# ─── Dependencies ─────────────────────────────────────────────────────────────
deps:
	go mod download
	cd $(WEB_DIR) && npm install

# ─── Tests ───────────────────────────────────────────────────────────────────
TEST_ENV = JWT_SECRET=00000000000000000000000000000000

test:
	$(TEST_ENV) go test -short -count=1 ./...

test-web:
	@echo "→ Running Vue unit tests..."
	cd $(WEB_DIR) && npx vitest run

test-cover:
	$(TEST_ENV) go test -short -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# ─── Clean ───────────────────────────────────────────────────────────────────
clean:
	rm -f $(API_NAME) $(MIGRATE) $(SEED) $(CLI_NAME)
	rm -rf $(WEB_DIR)/dist

# ─── Migrations ──────────────────────────────────────────────────────────────
MIGRATE_ENV = DATABASE_URL="postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable"

migrate:
	@echo "→ Running migrations..."
	$(MIGRATE_ENV) go run -buildvcs=false ./cmd/migrate -command=up

migrate-status:
	$(MIGRATE_ENV) go run -buildvcs=false ./cmd/migrate -command=status

migrate-down:
	$(MIGRATE_ENV) go run -buildvcs=false ./cmd/migrate -command=down

migrate-reset:
	$(MIGRATE_ENV) go run -buildvcs=false ./cmd/migrate -command=reset

# ─── Development ──────────────────────────────────────────────────────────────
DEV_DB_URL    ?= postgres://entsaas:secret-db-password@localhost:5432/entsaas?sslmode=disable
DEV_MASTER_KEY ?= 000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f
DEV_ADMIN_EMAIL ?= admin@entsaas.dev
DEV_ADMIN_PASS  ?= password

dev:
	@echo "Starting dev stack (Ctrl-C to stop)..."
	docker-compose --profile dev up --build

dev-up:
	@echo "Starting dev stack (detached)..."
	docker-compose --profile dev up --build -d --remove-orphans

dev-down:
	docker-compose --profile dev down

down: dev-down  ## Alias for dev-down

dev-logs:
	docker-compose logs -f api-dev web-dev

seed-dev:
	@echo "→ Seeding dev database..."
	DATABASE_URL="$(DEV_DB_URL)" \
	ENTSAAS_MASTER_KEY=$(DEV_MASTER_KEY) \
	ENTSAAS_ADMIN_EMAIL=$(DEV_ADMIN_EMAIL) \
	ENTSAAS_ADMIN_PASSWORD=$(DEV_ADMIN_PASS) \
	JWT_SECRET=00000000000000000000000000000001 \
	go run -buildvcs=false ./cmd/seed
	@echo "✅ Dev seed complete. Login: $(DEV_ADMIN_EMAIL) / $(DEV_ADMIN_PASS)"

dev-reseed:
	@echo "⚠️  Wiping and re-seeding dev environment..."
	docker-compose --profile dev down -v
	docker-compose --profile dev up --build -d
	@echo "→ Waiting 10s for databases..."
	sleep 10
	$(MAKE) migrate
	$(MAKE) seed-dev
	@echo "✅ Dev environment re-seeded."

# ─── Quality ─────────────────────────────────────────────────────────────────
lint:
	go vet ./...
	@echo "✅ Go lint passed."

check-local: test test-web lint
	@echo "✅ Local checks passed."

# ─── Doctor ────────────────────────────────────────────────────────────────
doctor:
	@echo "🔍 EntSaaS environment check"
	@command -v go   && echo "✅ Go: $$(go version)"   || echo "❌ Go not found"
	@command -v node && echo "✅ Node: $$(node --version)" || echo "❌ Node not found"
	@command -v npm  && echo "✅ npm: $$(npm --version)"   || echo "❌ npm not found"
	@command -v docker && echo "✅ Docker: $$(docker --version)" || echo "❌ Docker not found"
	@command -v docker-compose && echo "✅ docker-compose: $$(docker-compose --version)" || echo "❌ docker-compose not found"
	@test -f .env || (echo "⚠️  No .env file found. Copy .env.example to .env" && exit 1)
	@echo "✅ .env present"
	@grep -q DATABASE_URL .env && echo "✅ DATABASE_URL set" || echo "❌ DATABASE_URL missing in .env"
	@grep -q JWT_SECRET   .env && echo "✅ JWT_SECRET set"   || echo "❌ JWT_SECRET missing in .env"

# ─── Documentation ────────────────────────────────────────────────────────────
.PHONY: docs-deps docs-dev docs-audit docs-build docs-preview docs-docker

docs-deps:
	cd docs-site && npm install

docs-dev:
	cd docs-site && npm run docs:dev

docs-audit:
	cd docs-site && npm run docs:audit

docs-build:
	cd docs-site && npm run docs:build

docs-preview:
	cd docs-site && npm run docs:preview

docs-docker:
	docker build -f Dockerfile.docs -t entsaas-docs .

