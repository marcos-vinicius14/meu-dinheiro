---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# Codebase Concerns

**Analysis Date:** 2026-10-03

## Tech Debt

**Telegram Bot Command Handlers & State Machine:**
- Issue: In `apps/bot/internal/bot/bot.go`, only `/start`, `/ajuda`, and `/status` are implemented. Commands registered in BotFather (`/s2s`, `/gasto`, `/simular`, `/investimento`, `/checkin`) fall back to default "Comando não reconhecido". The multi-step conversational onboarding state machine (Milestone 2) is not yet wired.
- Files: `apps/bot/internal/bot/bot.go`
- Impact: Users attempting to use registered Telegram commands cannot perform daily operations via chat yet.
- Fix approach: Implement a dialogue state manager and message router in `apps/bot/internal/bot/` consuming `apps/bot/internal/client/api_client.go`.

**Float64 Usage in Bot API Client DTOs:**
- Issue: In `apps/bot/internal/client/api_client.go`, request and response structures use `float64` for amounts (e.g. `InitialBalance float64`, `S2SToday float64`), while the backend uses `money.Money` and `shopspring/decimal`.
- Files: `apps/bot/internal/client/api_client.go`
- Impact: Potential precision loss on extreme monetary quantities when serializing/deserializing between bot and API.
- Fix approach: Align bot DTOs with string-based or decimal representations matching `money.Money`.

**Hardcoded Dashboard Mock Values in Web App:**
- Issue: In `apps/web/src/App.vue`, the authenticated dashboard displays hardcoded KPI values (e.g., `R$ 4.250,00`, `R$ 91,66 / dia`) and quick action buttons trigger browser `alert()` dialogs rather than interacting with the API.
- Files: `apps/web/src/App.vue#L35-L62`
- Impact: Logged-in users on web do not see their real transactional balances or S2S metrics.
- Fix approach: Replace static mock markup with reactive state populated from `GET /transactions` and user financial context.

## Known Bugs

**None detected:**
- The active core domain logic, monetary arithmetic, and predictive algorithms pass all unit and integration tests without known runtime defects.

## Security Considerations

**Rate Limiting on Ephemeral Auth Endpoints:**
- Risk: `POST /auth/telegram/challenge` and `GET /auth/telegram/poll` do not have application-level rate limiting. An attacker could flood the API with challenge creation requests.
- Files: `apps/api/internal/auth/handler.go`
- Current mitigation: Challenges have a 5-minute TTL (`expires_at`), and status polling is protected by token uniqueness.
- Recommendations: Implement IP-based sliding-window rate limiting middleware or enforce rate limits at the reverse proxy (Nginx/Traefik).

**Internal API Secret Exposure:**
- Risk: Compromise of `INTERNAL_API_KEY` allows arbitrary authorization of challenges and financial mutations via `/internal/botapi/*`.
- Files: `apps/api/internal/botapi/handler.go`, `apps/bot/internal/config/config.go`
- Current mitigation: Verified via `X-Internal-Secret` or `X-Internal-API-Key` headers.
- Recommendations: Ensure `.env` is never committed; use mutual TLS or internal Docker networking in production.

## Performance Bottlenecks

**Challenge Table Growth:**
- Problem: `tb_telegram_auth_challenges` accumulates expired challenge records over time.
- Files: `apps/api/migrations/00001_initial_schema.sql`, `apps/api/internal/auth/handler.go`
- Cause: No periodic garbage collection or vacuuming worker deletes expired challenges.
- Improvement path: Introduce a recurring SQL cleanup task (`DELETE FROM tb_telegram_auth_challenges WHERE expires_at < NOW()`).

## Fragile Areas

**Docker Dependency for Test Execution:**
- Files: `apps/api/tests/integration/test_support.go`
- Why fragile: Running `make test` or `go test ./...` in `apps/api` requires a running Docker daemon to spin up Testcontainers (`postgres:18-alpine`). Environments without Docker or socket permissions fail tests immediately.
- Safe modification: Developers can run `cd apps/api && go test -v ./internal/...` for fast unit tests without Docker.

## Scaling Limits

**PostgreSQL Advisory Lock Limits on Account Creation:**
- Current capacity: High concurrency protection with `pg_advisory_xact_lock` based on `user_id` hash.
- Limit: Up to 3 bank accounts per user enforced defense-in-depth.
- Scaling path: Advisory locks are transaction-scoped and release on commit/rollback; scalable for single-tenant multi-user operation.

## Dependencies at Risk

**Go Version 1.27.0 in `go.mod`:**
- Risk: Both `apps/api/go.mod` and `apps/bot/go.mod` specify `go 1.27.0`. While synthetic/future-proofed in this environment, local toolchains must match Go version compatibility.
- Impact: Older Go toolchains (<1.22) will fail to compile.
- Migration plan: Maintain consistent Go compiler version across CI and local development.

## Missing Critical Features

**Telegram Bot Conversational Onboarding (Milestone 2):**
- Problem: The bot cannot guide first-time users through entering initial balance, cycle day, and essential expenses via Telegram chat.
- Blocks: Core loop completion for mobile-first users who do not configure via REST API directly.

**What-If Simulation Command (`/simular`):**
- Problem: Engine simulator exists in `apps/api/internal/transaction/engine/what_if_simulator.go`, but has no exposed Telegram command interface in `apps/bot`.
- Blocks: Instant pre-purchase affordability checks from mobile chat.

## Test Coverage Gaps

**Unit Tests for Telegram Bot Commands:**
- What's not tested: Message routing and conversational error handling in `apps/bot/internal/bot/bot.go`.
- Files: `apps/bot/internal/bot/bot.go`
- Risk: Regression in Telegram message parsing or reply generation.

---

*Concerns analysis: 2026-10-03*
