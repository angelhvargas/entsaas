# Developer Quickstart

Get your EntSaaS local development environment up and running in under five minutes.

---

## 1. Prerequisites

Before beginning, ensure you have the following installed on your machine:
- **Go** (v1.23 or higher)
- **Node.js** (v20 or higher) & **npm**
- **Docker** & **Docker Compose**
- **Make** (standard build tool)

---

## 2. Environment Setup

Clone the repository and copy the example environment configurations:

```bash
# Clone the repository (if not already done)
git clone https://github.com/your-username/entsaas.git
cd entsaas

# Copy the development environment file
cp .env.example .env.dev
```

Open `.env.dev` and configure your local parameters (defaults are set up out-of-the-box for single-command Docker launch).

---

## 3. Spin Up Services

Use the Makefile to build and start your PostgreSQL and Redis dependencies:

```bash
# Start Docker dependencies (PostgreSQL + Redis)
make compose-up
```

*This command launches the localized database containers in the background and maps port 5432 and 6379.*

---

## 4. Run Migrations & Hydrate Seeds

Synchronize your database structures and populate them with mock accounts, plans, and team workspaces:

```bash
# Run PostgreSQL migrations
make migrate-up

# Seed development mock data
make seed-db
```

This creates:
- The declarative billing catalog configurations.
- Two mock organizations with distinct subscription tiers (Free and Pro).
- Standard credentials for local login trials:
  - **Owner**: `owner@example.com` / `password123`
  - **Admin**: `admin@example.com` / `password123`
  - **Member**: `member@example.com` / `password123`

---

## 5. Launch Local Servers

### Spin Up Backend Go API:
```bash
# Start backend API (with live hot reload enabled)
make run
```
*The API is now running at `http://localhost:8080`.*

### Spin Up Frontend SPA:
Open a second terminal window and start the Vue client dev environment:
```bash
cd web
npm install
npm run dev
```
*The frontend SPA client is now running at `http://localhost:5173`.*

Open your browser and navigate to `http://localhost:5173` to log in, test project scopes, send team invites, check dynamic billing checkouts, and execute streaming AI chat tests!
