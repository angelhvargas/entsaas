# CLI Administration

EntSaaS features a built-in admin CLI command utility called `entsaasctl` located at `/cmd/entsaasctl`. This utility provides a secure interface for executing administrative tasks on live production environments.

---

## 1. CLI Commands Index

### 1.1 Database Migration
Manages goose schema updates:
```bash
# Apply all pending schema migrations
./entsaasctl migrate up

# Rollback the last migration
./entsaasctl migrate down

# Check migration status
./entsaasctl migrate status
```

### 1.2 Catalog Synchronization
Reconciles the database catalog with the declarative YAML config:
```bash
# Sync billing.yaml plans, prices, and entitlements to the DB
./entsaasctl billing sync
```

### 1.3 Seeding Environment Data
Populates databases with mock accounts and premium tiers:
```bash
# Seed development mock organizations, users, and plans
./entsaasctl seed dev
```

### 1.4 Managing Team Invitations
Provides administrative controls to inspect and revoke outstanding org invitations:
```bash
# List all active team invitations
./entsaasctl invite list

# Revoke a team invitation by token
./entsaasctl invite revoke --token="invite_token_here"
```

---

## 2. Compiling the CLI

To compile the administrative CLI utility, run:
```bash
# Compiles entsaasctl and saves it in the root folder
go build -buildvcs=false -o entsaasctl cmd/entsaasctl/main.go
```
