---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# External Integrations

**Analysis Date:** 2026-10-03

## APIs & External Services

**Telegram Bot API:**
- Telegram messaging platform: powers the 1-click authentication challenge flow, conversational onboarding, and daily operations (`/start`, `/ajuda`, `/status`, and planned `/s2s`, `/gasto`, `/simular`, `/investimento`).
  - SDK/Client: `github.com/go-telegram-bot-api/telegram-bot-api/v5` in `apps/bot/internal/bot/bot.go`
  - Auth: `TELEGRAM_BOT_TOKEN`
  - Integration modes: Long Polling (development default) or HTTPS Webhook (`apps/bot/internal/bot/webhook.go`)

**Internal REST API (Bot ↔ API):**
- HTTP service layer connecting `apps/bot` to `apps/api`:
  - Client: `apps/bot/internal/client/api_client.go`
  - Endpoints in `apps/api/internal/botapi/handler.go`:
    - `POST /internal/auth/authorize-challenge`
    - `POST /internal/users/onboarding`
    - `GET /internal/users/context-by-telegram`
    - `POST /internal/investments`
    - `POST /internal/transactions/quick-expense`
  - Auth: Shared secret via HTTP headers `X-Internal-Secret` or `X-Internal-API-Key` (`INTERNAL_API_KEY`)

## Data Storage

**Databases:**
- PostgreSQL 18 (Alpine image `postgres:18-alpine` in `compose.yaml`)
  - Connection: `DATABASE_URL` (e.g. `postgres://marcos:local_db@localhost:5432/meu_dinheiro?sslmode=disable`)
  - Client: `jackc/pgx/v5/pgxpool` in `apps/api/internal/database/database.go`
  - Migrations: `pressly/goose/v3` embedded in Go binary (`apps/api/migrations/migrations.go`)
  - Native features used: UUIDv7 (`uuidv7()`), advisory transactional locks (`pg_advisory_xact_lock`), foreign key cascades, unique constraints

**File Storage:**
- Local filesystem only (`embed.FS` for embedded SQL migrations; static asset bundling for web via Vite)

**Caching:**
- None / In-database transactional snapshots (`tb_check_in_snapshots`)

## Authentication & Identity

**Auth Provider:**
- Custom Telegram-First Passwordless Authentication:
  - Approach: Ephemeral cryptographic challenge tokens with 5-minute TTL (`tb_telegram_auth_challenges`)
  - Browser requests challenge: `POST /auth/telegram/challenge`
  - Deep link format: `tg://resolve?domain=<BOT_USERNAME>&start=auth_<token>`
  - Bot validates challenge via internal endpoint: `POST /internal/auth/authorize-challenge`
  - Browser polls: `GET /auth/telegram/poll?token=<token>`
  - Session established via HTTP-only, SameSite cookie (`session_token`) storing an HMAC-SHA256 JWT
  - User identity anchored to unique Telegram ID (`telegram_id BIGINT UNIQUE` in `tb_users`)

## Monitoring & Observability

**Error Tracking:**
- None / Standard error logging

**Logs:**
- Structured logging using Go's standard library `log/slog` in `apps/bot` (`apps/bot/cmd/bot/main.go`) and standard `log` in `apps/api`
- HTTP request/response logging middleware in `apps/api/internal/web/web.go` (`Logger` middleware)

## CI/CD & Deployment

**Hosting:**
- Self-hosted on Coolify / VPS Docker Compose
- Development database via Docker Compose (`compose.yaml`)

**CI Pipeline:**
- GitHub Actions workflow defined in `.github/workflows/` (running tests and builds)

## Environment Configuration

**Required env vars:**
- `PORT`: API server listening port (e.g. `8080`)
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Cryptographic secret for signing session cookies
- `BOT_USERNAME`: Telegram handle without '@' for deep linking
- `INTERNAL_API_KEY`: Secret key shared between bot and API
- `TELEGRAM_BOT_TOKEN`: Telegram bot token from @BotFather
- `API_BASE_URL`: REST API address for the bot (e.g. `http://localhost:8080`)

**Secrets location:**
- Stored in root `.env` (gitignored, template provided in `.env.example`)

## Webhooks & Callbacks

**Incoming:**
- Telegram Webhook: `POST /webhook` (optional production mode in `apps/bot/internal/bot/webhook.go`, protected by `X-Telegram-Bot-Api-Secret-Token`)
- Telegram Polling Poll: `GET /auth/telegram/poll` in `apps/api/internal/auth/handler.go`

**Outgoing:**
- Telegram API calls to `https://api.telegram.org/bot<TOKEN>/...` via `go-telegram-bot-api`

---

*Integration audit: 2026-10-03*
