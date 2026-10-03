---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# Coding Conventions

**Analysis Date:** 2026-10-03

## Naming Patterns

**Files:**
- Go: lowercase `snake_case.go` (e.g. `predictive_engine.go`, `date_interval.go`, `api_client.go`)
- Go Tests: `*_test.go` matching the source file (e.g. `money_test.go`, `webhook_test.go`)
- Vue SFCs: `PascalCase.vue` (e.g. `App.vue`)
- Migrations: `NNNNN_snake_case_description.sql` (e.g. `00001_initial_schema.sql`)

**Functions & Methods:**
- Go: idiomatic `camelCase` for unexported functions and `PascalCase` for exported functions
- Factory constructors: `New<Type>()` (e.g. `NewPredictiveEngine()`, `NewAPIClient()`, `NewRepository(pool)`)
- Short, contextual naming avoiding repetition (e.g. `user.NewRepository()`, not `user.NewUserRepository()`)
- Preserved standard technical acronyms: `ID`, `UUID`, `HTTP`, `API`, `SQL`, `JWT`, `S2S`

**Variables:**
- Go: short receiver names (1-2 letters, e.g. `(h *Handler)`, `(r *Repository)`, `(m Money)`)
- Context variables always named `ctx`
- Error variables always named `err`
- Deserialization targets named `req` and serialization outputs named `resp`

**Types:**
- Go structs and interfaces: `PascalCase` (e.g. `PredictiveEngine`, `CycleContext`, `BankAccount`, `Transaction`)
- Sentinel errors: `Err<Reason>` (e.g. `ErrUserNotFound`, `ErrInvalidTicker`, `ErrAccountLimitReached`)

## Code Style

**Formatting:**
- Go: `gofmt -s -w .` and `go vet ./...` strictly adhered to
- Tabs for indentation in Go; 2 spaces in JavaScript / Vue / YAML
- Double quotes for Go strings and JSON; single quotes in Vue template attributes when appropriate

**Linting:**
- Go compiler and `go vet` for static analysis
- Strict prohibition of unused variables and imports

## Import Organization

**Order (Go):**
1. Standard library packages (e.g. `context`, `fmt`, `net/http`, `time`)
2. External third-party packages (e.g. `github.com/go-chi/chi/v5`, `github.com/shopspring/decimal`)
3. Internal module packages (e.g. `github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/...`)

**Path Aliases:**
- Standard Go full module path imports (`github.com/marcos-vinicius14/meu-dinheiro/apps/api/...`)
- No relative imports (`../`) across Go packages

## Error Handling

**Language & Tone (pt-BR):**
- All user-facing error messages, HTTP response bodies, and sentinel descriptions **MUST be in Portuguese (pt-BR)** (e.g. `"usuário não encontrado"`, `"limite máximo de 3 contas atingido"`, `"valor do gasto deve ser maior que zero"`).
- Internal code identifiers and error constants remain in English (`ErrUserNotFound`, `ErrInvalidTicker`).

**Error Wrapping:**
- Errors always propagated and wrapped using `%w`:
  ```go
  if err != nil {
      return fmt.Errorf("buscar usuário por telegram: %w", err)
  }
  ```
- Sentinel errors verified using `errors.Is(err, targetErr)`.

**HTTP Error Translation:**
- Standardized via `web.Error(w, statusCode, message)`:
  ```go
  if err == user.ErrUserNotFound {
      web.Error(w, http.StatusNotFound, "Usuário não encontrado")
      return
  }
  ```

## Logging

**Framework:**
- Go standard `log/slog` for structured logging in `apps/bot`
- Standard `log` for lifecycle logging in `apps/api`
- HTTP access logging middleware in `apps/api/internal/web/web.go`

**Patterns:**
- Key-value attributes in logs (no unstructured `fmt.Sprintf` for variable data):
  ```go
  logger.Info("mensagem recebida", "chat_id", chatID, "user_id", user.ID)
  ```
- Never log sensitive tokens (e.g., Telegram bot tokens, full JWT secret, database credentials).

## Comments

**When to Comment:**
- Business logic rationales, financial formula rules, and non-obvious algorithms
- Step-by-step numbering in complex orchestrations (e.g., in `handleOnboarding` and `handleQuickExpense`)

**JSDoc/TSDoc & GoDoc:**
- Top-level exported types and methods should include clear GoDoc explanations when behavior is domain-specific.

## Function Design

**Size:**
- Small, focused functions. Handlers delegate to domain services or repositories.
- Algorithmic engines split into clear mathematical passes (e.g. `CalculateCommittedExpenses`, `AssessHealth`).

**Parameters:**
- `context.Context` is **always** the first parameter (`ctx context.Context`) for any function performing I/O.
- Value semantics for lightweight immutable objects (`money.Money`, `dateinterval.DateInterval`).
- Pointer semantics for structs holding state or large fields (`*user.Repository`, `*transaction.Transaction`).

**Return Values:**
- Idiomatic Go `(result, error)` tuples. Never return untyped `interface{}` where concrete types are known.

## Module Design

**Exports:**
- Minimal exported surface area: keep internal helper functions, SQL statements, and DTOs unexported unless needed across packages.
- Strict package separation by domain (`internal/bankaccount`, `internal/investment`, `internal/transaction`).

---

*Convention analysis: 2026-10-03*
