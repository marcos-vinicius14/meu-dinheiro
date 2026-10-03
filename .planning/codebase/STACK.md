---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# Technology Stack

**Analysis Date:** 2026-10-03

## Languages

**Primary:**
- Go 1.27.0 - Backend services (`apps/api` and `apps/bot`)
- JavaScript / ECMAScript (ES modules) - Frontend SPA (`apps/web`)

**Secondary:**
- SQL (PostgreSQL 18 dialect) - Database migrations (`apps/api/migrations/*.sql`)
- HTML5 / CSS3 - UI views and styling (`apps/web/src/App.vue`, `apps/web/index.html`)

## Runtime

**Environment:**
- Go 1.27 runtime (Linux x86_64 native)
- Node.js runtime for frontend development and build tooling (Vite)
- Docker / Docker Compose engine for database service (`postgres:18-alpine`)

**Package Manager:**
- Go Modules (`go.mod` / `go.sum` in `apps/api` and `apps/bot`)
  - Lockfile: `apps/api/go.sum` and `apps/bot/go.sum` present
- npm for web application (`apps/web/package.json`)
  - Lockfile: `apps/web/package-lock.json` present

## Frameworks

**Core:**
- `go-chi/chi/v5` (v5.3.2) - Idiomatic and lightweight HTTP router for `apps/api`
- `go-telegram-bot-api/telegram-bot-api/v5` (v5.5.1) - Telegram Bot API wrapper in `apps/bot`
- `vue` (v3.5.13) - Reactive frontend Single-Page Application (Composition API with `<script setup>`)

**Testing:**
- Go standard `testing` package - Unit and integration test runner
- `stretchr/testify` (v1.12.1) - Assertion and test suite helpers (`assert`, `require`)
- `testcontainers/testcontainers-go` (v0.44.0) with module `testcontainers-go/modules/postgres` - Spins up real PostgreSQL 18 containers for integration testing (`apps/api/tests/integration/test_support.go`)

**Build/Dev:**
- `vite` (v6.2.0) - Next-generation frontend bundler and dev server (`apps/web/vite.config.js`)
- `@vitejs/plugin-vue` (v5.2.1) - Vue 3 Single File Component compiler plugin
- `pressly/goose/v3` (v3.28.0) - SQL database migrations with Go embedded file system (`embed.FS`)
- GNU Make - Top-level task runner (`Makefile`)

## Key Dependencies

**Critical:**
- `shopspring/decimal` (v1.4.0) - Arbitrary-precision fixed-point decimal arithmetic for monetary operations, avoiding floating-point inaccuracies with `RoundBank` (HalfEven rounding) (`apps/api/internal/money/money.go`)
- `golang-jwt/jwt/v5` (v5.3.1) - HMAC-SHA256 JWT generation, signing, and cookie-based authentication validation (`apps/api/internal/auth/jwt.go`)
- `google/uuid` (v1.6.0) - UUID generation and parsing (paired with PostgreSQL `uuidv7()`)

**Infrastructure:**
- `jackc/pgx/v5` (v5.11.0) and `jackc/puddle/v2` (v2.2.2) - High-performance PostgreSQL driver and connection pooling (`pgxpool.Pool`) (`apps/api/internal/database/database.go`)
- `log/slog` (Go standard library) - Structured logging across `apps/bot` and API services

## Configuration

**Environment:**
- Configured via environment variables loaded from `.env` file (see `.env.example`)
- In `apps/api`: parsed in `apps/api/internal/config/config.go`
  - `PORT`: HTTP port (default `8080`)
  - `DATABASE_URL`: PostgreSQL connection string (`postgres://...`)
  - `JWT_SECRET`: Secret key for session JWT signing
  - `BOT_USERNAME`: Telegram bot handle for deep links
  - `INTERNAL_API_KEY`: Shared secret for internal bot ↔ API communication
- In `apps/bot`: parsed in `apps/bot/internal/config/config.go`
  - `TELEGRAM_BOT_TOKEN`: Bot token from @BotFather
  - `API_BASE_URL`: REST API base URL (default `http://localhost:8080`)
  - `INTERNAL_API_KEY`: Shared secret passed via `X-Internal-Secret` header
  - `WEBHOOK_URL`, `WEBHOOK_SECRET_TOKEN`, `WEBHOOK_PORT`, `WEBHOOK_PATH`: Webhook configuration (Long Polling if empty)

**Build:**
- Top-level `Makefile`: targets `test`, `test-api`, `test-bot`, `run-api`, `run-bot`, `run-web`, `build-all`, `up`, `down`
- `compose.yaml`: PostgreSQL 18 Alpine container configuration
- `apps/web/vite.config.js`: Vite dev server configuration

## Platform Requirements

**Development:**
- Linux (or macOS / WSL2)
- Go 1.27+
- Node.js 20+ and npm
- Docker Engine & Docker Compose (required for running tests with Testcontainers and local PostgreSQL)

**Production:**
- Containerized Linux environment or Coolify VPS
- PostgreSQL 18 database with `uuidv7()` support
- Outbound network access for Telegram Bot API communication (`api.telegram.org`)
- Inbound HTTPS for web application and optional Telegram Webhook

---

*Stack analysis: 2026-10-03*
