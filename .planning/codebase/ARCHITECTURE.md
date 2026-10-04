---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
<!-- refreshed: 2026-10-03 -->

# Architecture

**Analysis Date:** 2026-10-03

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                 Client Layer (UI & Chat)                    │
├──────────────────────────────┬──────────────────────────────┤
│    Vue 3 Web Application     │      Telegram Messenger      │
│       `apps/web/src/`        │      (End-User Mobile/Web)   │
└──────────────┬───────────────┴──────────────┬───────────────┘
               │                              │
               │ HTTP / JSON                  │ Telegram Bot API
               ▼                              ▼
┌──────────────────────────────┐ ┌────────────────────────────┐
│      API Gateway (Chi)       │ │     Telegram Bot Daemon    │
│    `apps/api/cmd/api/`       │ │    `apps/bot/cmd/bot/`     │
├──────────────────────────────┤ └─────────────┬──────────────┘
│ internal/web: router, cors   │               │
│ internal/auth: JWT cookies   │               │ Internal HTTP (Secret Key)
└──────────────┬───────────────┘               ▼
               │                   `POST /internal/botapi/*`
               ▼
┌─────────────────────────────────────────────────────────────┐
│                       Domain Services                        │
├──────────────────────────────┬──────────────────────────────┤
│  transaction.Service         │  investment.Service          │
│  `apps/api/internal/`        │  `apps/api/internal/`        │
├──────────────────────────────┴──────────────────────────────┤
│  engine.PredictiveEngine: S2S Daily Allowance & Health       │
│  engine.WhatIfSimulator: 12-Cycle Multi-Cycle Forecast      │
│  dateinterval.CycleInterval: Dynamic Payroll Cycles         │
│  money.Money: Banker's Rounding HalfEven Fixed Arithmetic    │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                  Persistence & Database                     │
│  PostgreSQL 18 (`tb_users`, `tb_transactions`, etc.)        │
│  `apps/api/internal/database/` with `jackc/pgx/v5` Pool     │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| `cmd/api` | API entry point, database connection setup, migrations execution, graceful shutdown | `apps/api/cmd/api/main.go` |
| `cmd/bot` | Telegram bot entry point, polling/webhook lifecycle, signal interception | `apps/bot/cmd/bot/main.go` |
| `auth` | Telegram challenge creation, polling, token authorization, JWT cookie management | `apps/api/internal/auth/handler.go` |
| `botapi` | Internal API consumed by the Telegram Bot for onboarding, context, investments, quick expenses | `apps/api/internal/botapi/handler.go` |
| `bankaccount` | Bank account CRUD with transactional locks and defensive 3-account limit | `apps/api/internal/bankaccount/repository.go` |
| `category` | Income/expense categories with case-insensitive unique naming | `apps/api/internal/category/repository.go` |
| `transaction` | Transaction CRUD, installment bundles, check-in snapshots | `apps/api/internal/transaction/service.go` |
| `engine` | Deterministic Daily Safe-to-Spend (S2S) computation & 12-cycle what-if simulation | `apps/api/internal/transaction/engine/predictive_engine.go` |
| `investment` | Stock portfolio, weighted average purchase price calculation, position sales | `apps/api/internal/investment/service.go` |
| `user` | User provisioning via Telegram ID, financial profile & emergency fund settings | `apps/api/internal/user/repository.go` |
| `money` | Immutable monetary representation with Banker's rounding (`decimal.Decimal`) | `apps/api/internal/money/money.go` |
| `dateinterval` | Civil months, custom payroll cycles (day 1-28), inclusive remaining day calculations | `apps/api/internal/dateinterval/date_interval.go` |
| `web` | Chi router configuration, JSON response helpers, recovery and logging middleware | `apps/api/internal/web/web.go` |
| `client` | HTTP client used by the Telegram bot to invoke the backend API | `apps/bot/internal/client/api_client.go` |
| `bot` | Telegram update processing, command dispatcher, message rendering | `apps/bot/internal/bot/bot.go` |

## Pattern Overview

**Overall:** Domain-Driven Modular Monolith in Go with idiomatic standard library conventions, transactional repository pattern, and zero-ORM raw SQL queries via `pgx/v5`.

**Key Characteristics:**
- Strict package-by-domain isolation (`internal/user`, `internal/transaction`, `internal/investment`)
- Pure algorithmic domains (`money`, `dateinterval`, `engine`) completely decoupled from HTTP and database I/O
- Deterministic predictive mathematical models for financial projection
- Telegram-first authentication replacing traditional password/email/BCrypt models

## Layers

**HTTP / Web Layer:**
- Purpose: HTTP routing, parameter deserialization, validation, status code mapping, cookie management
- Location: `apps/api/internal/web/`, `apps/api/internal/auth/handler.go`, `apps/api/internal/botapi/handler.go`
- Contains: Chi route declarations, HTTP handlers, middleware
- Depends on: Domain services and repositories
- Used by: External clients (Vue 3 Web SPA, Telegram Bot daemon)

**Domain Service Layer:**
- Purpose: Orchestrate business transactions, coordinate multiple repositories, enforce business rules
- Location: `apps/api/internal/transaction/service.go`, `apps/api/internal/investment/service.go`, `apps/api/internal/auth/service.go`
- Contains: Business workflows, mathematical validation
- Depends on: Repositories and engine calculators
- Used by: HTTP Handlers

**Calculation Engine (Pure Logic):**
- Purpose: Compute Safe-to-Spend (S2S), cycle calendar arithmetic, weighted average prices, what-if simulations
- Location: `apps/api/internal/transaction/engine/`, `apps/api/internal/money/`, `apps/api/internal/dateinterval/`
- Contains: Pure Go functions and structs, zero database dependencies, 100% unit-tested
- Depends on: Standard library and `shopspring/decimal`
- Used by: Handlers and domain services

**Persistence / Repository Layer:**
- Purpose: Execute raw parameterized SQL queries against PostgreSQL 18
- Location: `apps/api/internal/*/repository.go`
- Contains: Explicit column queries, transaction isolation, advisory locks
- Depends on: `jackc/pgx/v5` and `pgxpool.Pool`
- Used by: Domain services and handlers

## Data Flow

### Primary Request Path: Telegram 1-Click Authentication

1. User clicks "Entrar com Telegram" in Vue Web (`apps/web/src/App.vue#L189-L216`)
2. Web invokes `POST /auth/telegram/challenge` (`apps/api/internal/auth/handler.go#L30`)
3. API creates ephemeral challenge in `tb_telegram_auth_challenges` and returns deep link
4. Web opens `tg://resolve?domain=meu_dinheiro_bot&start=auth_<token>` and polls `GET /auth/telegram/poll?token=<token>`
5. User taps "Start" in Telegram; Bot handles `/start auth_<token>` (`apps/bot/internal/bot/bot.go#L89-L96`)
6. Bot calls `POST /internal/auth/authorize-challenge` with `X-Internal-Secret` (`apps/bot/internal/client/api_client.go#L35`)
7. API verifies token, finds or provisions user in `tb_users`, updates challenge status to `AUTHORIZED`
8. Next poll by browser succeeds, setting HTTP-only `session_token` cookie containing signed JWT (`apps/api/internal/auth/jwt.go`)

### Secondary Flow: Daily Safe-to-Spend (S2S) Calculation

1. Request arrives via Web dashboard or Telegram Bot (`/s2s` or `GET /internal/users/context-by-telegram`)
2. Context fetches current liquid account balance and cycle parameters (`apps/api/internal/botapi/handler.go#L341-L352`)
3. Determines cycle boundaries via `dateinterval.CycleOf(currentDate, cycleStartDay)`
4. Loads transaction snapshots within the active cycle (`apps/api/internal/transaction/repository.go`)
5. `engine.PredictiveEngine.Calculate` runs deterministic formula:
   - Computes committed expenses, projected revenues, and flexible spending already realized
   - Derives remaining flexible liquidity and divides by remaining days in the cycle
   - Assesses cycle health: `HEALTHY`, `RESTRICTED`, or `DEFICIT_RISK`
6. Formats response with Banker's rounded currency amounts

**State Management:**
- Client-side (Web): Reactive Vue 3 `ref` state (`apps/web/src/App.vue`)
- Server-side: Stateless HTTP request handling; session stored in JWT cookie; persistence strictly in PostgreSQL 18

## Key Abstractions

**Money:**
- Purpose: Absolute precision monetary value wrapper around `decimal.Decimal`
- Examples: `apps/api/internal/money/money.go`
- Pattern: Value Object with immutable arithmetic methods (`Add`, `Subtract`, `Multiply`, `Allocate`, `RoundBank`)

**DateInterval & CycleInterval:**
- Purpose: Immutable calendar date ranges representing payroll cycles
- Examples: `apps/api/internal/dateinterval/date_interval.go`
- Pattern: Value Object with inclusive date arithmetic and clamping (days 1-28)

**PredictiveEngine:**
- Purpose: Deterministic mathematical model for daily allowances and bottleneck alerts
- Examples: `apps/api/internal/transaction/engine/predictive_engine.go`
- Pattern: Pure functional calculation strategy

## Entry Points

**API Server:**
- Location: `apps/api/cmd/api/main.go`
- Triggers: `make run-api` or running compiled binary `./bin/api`
- Responsibilities: Migrations execution, pgx pool initialization, route registration, HTTP listening on `:8080`, graceful termination on `SIGINT`/`SIGTERM`

**Telegram Bot:**
- Location: `apps/bot/cmd/bot/main.go`
- Triggers: `make run-bot` or running compiled binary `./bin/bot`
- Responsibilities: Long polling or webhook listener for Telegram updates, command parsing, API communication

**Web Frontend:**
- Location: `apps/web/src/main.js` and `apps/web/index.html`
- Triggers: `npm run dev` or production build serving
- Responsibilities: Mounts Vue 3 root application, manages auth modal and dashboard rendering

## Architectural Constraints

- **Precision:** No IEEE 754 floating-point numbers (`float32`/`float64`) for money in `apps/api`. Always `money.Money` (`decimal.Decimal`).
- **Concurrency & Locking:** Resource-capped modifications (e.g. max 3 bank accounts) and category uniqueness under concurrent writes use PostgreSQL transactional advisory locks (`pg_advisory_xact_lock`).
- **Testing Fidelity:** Zero database mocks (`sqlmock` is forbidden). Integration tests must execute against real PostgreSQL 18 instances via Testcontainers.
- **Security:** Telegram bot endpoints protected by shared secret (`X-Internal-Secret` / `X-Internal-API-Key`). User web sessions secured via HTTP-only, SameSite cookies.

## Anti-Patterns

### Using Float for Currency

**What happens:** Rounding errors (e.g., `0.1 + 0.2 = 0.30000000000000004`) distorting daily S2S and account reconciliation.
**Why it's wrong:** Breaks user trust and violates banking standards.
**Do this instead:** Use `money.Money` from `apps/api/internal/money/money.go`.

### Database Query in Loop (N+1)

**What happens:** Querying transactions or accounts individually inside an iteration.
**Why it's wrong:** Degrades database performance and exhausts connection pools.
**Do this instead:** Batch fetch via SQL aggregations or load snapshots in a single query (`apps/api/internal/transaction/repository.go`).

## Error Handling

**Strategy:** Explicit error inspection and propagation with `%w` wrapping in Go.

**Patterns:**
- Typed sentinel errors: `user.ErrUserNotFound`, `bankaccount.ErrAccountLimitReached`, `category.ErrCategoryExists`, `investment.ErrInvalidTicker`
- Translation to Portuguese HTTP JSON error responses: `web.Error(w, http.StatusBadRequest, "mensagem amigável")`

## Cross-Cutting Concerns

**Logging:**
- Standard structured logging via `log/slog` in `apps/bot` and HTTP request logging middleware in `apps/api/internal/web/web.go`

**Validation:**
- Invariant domain checks in constructors/methods; HTTP payload structural validation in handlers

**Authentication:**
- Middleware `auth.RequireAuth` verifies JWT from `session_token` cookie or bearer token, injects user context into `r.Context()`

---

*Architecture analysis: 2026-10-03*
