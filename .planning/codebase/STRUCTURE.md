---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# Codebase Structure

**Analysis Date:** 2026-10-03

## Directory Layout

```
meu-dinheiro-api/
├── Makefile                                # Top-level build and test automation targets
├── compose.yaml                            # Docker Compose definition (PostgreSQL 18 Alpine)
├── README.md                               # Project presentation and onboarding instructions
├── AGENTS.md                               # Canonical agent instructions and architectural rules
├── GEMINI.md                               # Gemini / Antigravity specialized rules
├── .env.example                            # Template of required environment variables
├── .github/                                # GitHub Actions CI workflows
│   └── workflows/
├── docs/                                   # Product documentation and roadmap
│   └── roadmap.md                          # Milestone roadmap and implementation tracker
├── bin/                                    # Compiled Go binary output directory (gitignored)
├── apps/                                   # Monorepo applications directory
│   ├── api/                                # RESTful Backend API written in Go
│   │   ├── cmd/api/                        # API daemon entry point (main.go)
│   │   ├── migrations/                     # Embedded SQL database migrations (Goose)
│   │   ├── internal/                       # Internal private domain packages
│   │   │   ├── auth/                       # Telegram challenge & JWT cookie auth
│   │   │   ├── bankaccount/                # Bank accounts domain & advisory locking
│   │   │   ├── botapi/                     # Dedicated endpoints consumed by Telegram Bot
│   │   │   ├── category/                   # Transaction categories domain
│   │   │   ├── config/                     # Environment variable parser
│   │   │   ├── database/                   # pgxpool connection & Goose migrations runner
│   │   │   ├── dateinterval/               # Date arithmetic & payroll cycle boundaries
│   │   │   ├── investment/                 # Stock portfolio & weighted average price
│   │   │   ├── money/                      # Banker's rounding decimal money abstraction
│   │   │   ├── transaction/                # Transactions, bundles, check-ins
│   │   │   │   └── engine/                 # S2S predictive engine & what-if simulator
│   │   │   ├── user/                       # User profile, Telegram ID anchor, financial goals
│   │   │   └── web/                        # Chi HTTP router, middleware, JSON responder
│   │   ├── tests/                          # Integration testing suite
│   │   │   └── integration/                # Testcontainers PostgreSQL 18 E2E test suite
│   │   ├── go.mod                          # Go module definition for api
│   │   └── go.sum                          # Go dependency lockfile
│   ├── bot/                                # Telegram Bot daemon written in Go
│   │   ├── cmd/bot/                        # Bot daemon entry point (main.go)
│   │   ├── internal/                       # Internal private bot packages
│   │   │   ├── bot/                        # Telegram update processing, polling & webhook
│   │   │   ├── client/                     # Internal HTTP client for apps/api
│   │   │   └── config/                     # Bot environment configuration
│   │   ├── go.mod                          # Go module definition for bot
│   │   └── go.sum                          # Go dependency lockfile
│   └── web/                                # Frontend Single-Page Application (Vue 3 + Vite)
│       ├── index.html                      # HTML root template
│       ├── package.json                    # Node.js project manifest & scripts
│       ├── vite.config.js                  # Vite build configuration
│       ├── dist/                           # Compiled production assets
│       └── src/                            # Vue 3 source code
│           ├── main.js                     # Vue app initialization & mounting
│           ├── App.vue                     # Main App component (Hero, Dashboard, Auth Modal)
│           └── views/                      # Dedicated view components
```

## Directory Purposes

**`apps/api`:**
- Purpose: Core backend REST API providing business logic, deterministic financial engines, and persistence.
- Contains: Go source files, embedded Goose SQL migrations, Testcontainers integration suite.
- Key files: `apps/api/cmd/api/main.go`, `apps/api/internal/transaction/engine/predictive_engine.go`, `apps/api/internal/botapi/handler.go`.

**`apps/bot`:**
- Purpose: Telegram interface daemon acting as the user-facing operational surface.
- Contains: Telegram Bot API handlers, long polling/webhook handlers, internal API client.
- Key files: `apps/bot/cmd/bot/main.go`, `apps/bot/internal/bot/bot.go`, `apps/bot/internal/client/api_client.go`.

**`apps/web`:**
- Purpose: Web application and Landing Page offering 1-click Telegram login and financial dashboard.
- Contains: Vue 3 Single-File Components (`.vue`), Vite configuration, CSS styling.
- Key files: `apps/web/src/App.vue`, `apps/web/vite.config.js`.

**`docs`:**
- Purpose: Project documentation and roadmapping.
- Key files: `docs/roadmap.md`.

## Key File Locations

**Entry Points:**
- API Daemon: `apps/api/cmd/api/main.go`
- Telegram Bot: `apps/bot/cmd/bot/main.go`
- Web Frontend: `apps/web/src/main.js`

**Configuration:**
- Environment Template: `.env.example`
- API Config: `apps/api/internal/config/config.go`
- Bot Config: `apps/bot/internal/config/config.go`
- Web Config: `apps/web/vite.config.js`
- Docker Compose: `compose.yaml`

**Core Logic & Engines:**
- Safe-to-Spend (S2S) Engine: `apps/api/internal/transaction/engine/predictive_engine.go`
- 12-Cycle What-If Simulator: `apps/api/internal/transaction/engine/what_if_simulator.go`
- Decimal Monetary Operations: `apps/api/internal/money/money.go`
- Date & Cycle Calculations: `apps/api/internal/dateinterval/date_interval.go`
- Investment Lot Calculations: `apps/api/internal/investment/investment.go`

**Testing:**
- Integration Test Base: `apps/api/tests/integration/test_support.go`
- Predictive Engine Tests: `apps/api/internal/transaction/engine/predictive_engine_test.go`
- Bot Client Tests: `apps/bot/internal/client/api_client_test.go`

## Naming Conventions

**Files:**
- Go files: `snake_case.go` (e.g. `bank_account.go`, `date_interval.go`, `predictive_engine.go`)
- Go test files: `*_test.go` co-located with implementation (or under `tests/integration/` for db tests)
- SQL Migrations: Sequential zero-padded format `00001_initial_schema.sql`, `00002_create_investments_and_user_financial_profile.sql`
- Vue files: `PascalCase.vue` (e.g. `App.vue`)

**Directories:**
- Package directories: Single lowercase word or condensed (e.g. `bankaccount`, `dateinterval`, `botapi`)

## Where to Add New Code

**New Financial Calculation / Engine Rule:**
- Implementation: `apps/api/internal/transaction/engine/`
- Unit tests: Co-located in `apps/api/internal/transaction/engine/*_test.go`

**New API Endpoint for Web / Public:**
- Domain package: Create or extend `apps/api/internal/<domain>/`
- Routing registration: In `apps/api/cmd/api/main.go`
- Integration tests: `apps/api/tests/integration/<domain>_test.go`

**New Telegram Bot Feature / Command:**
- Bot message handling: `apps/bot/internal/bot/bot.go`
- Internal API client call: `apps/bot/internal/client/api_client.go`
- API backend endpoint (if needed): `apps/api/internal/botapi/handler.go`

**New Database Migration:**
- SQL file: `apps/api/migrations/<NNNNN>_<description>.sql` with `-- +goose Up` and `-- +goose Down` sections

## Special Directories

**`.planning/`:**
- Purpose: GSD workflow state, codebase documentation, and roadmap metadata
- Generated: Yes (by GSD tooling)
- Committed: Yes

**`bin/`:**
- Purpose: Built Go binaries (`api`, `bot`)
- Generated: Yes (by `make build-all`)
- Committed: No (gitignored)

**`apps/web/dist/`:**
- Purpose: Built static assets for frontend
- Generated: Yes (by `npm run build`)
- Committed: No (gitignored)

---

*Structure analysis: 2026-10-03*
