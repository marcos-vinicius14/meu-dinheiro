---
last_mapped_commit: 2b2fb4e19fe6c14f50c5a613f66620de6f9622eb
last_mapped_at: 2026-10-03
---
# Testing Patterns

**Analysis Date:** 2026-10-03

## Test Framework

**Runner:**
- Standard Go test runner (`go test`)
- Parallelism and race detection: `go test -v -race ./...`

**Assertion Library:**
- `github.com/stretchr/testify/assert` and `github.com/stretchr/testify/require` (v1.12.1)

**Run Commands:**

```bash
make test              # Runs all test suites (apps/api and apps/bot)
make test-api          # Runs API unit and Testcontainers integration tests
make test-bot          # Runs Telegram bot unit and webhook tests
cd apps/api && go test -v ./internal/...  # Fast unit tests only (without Docker)
```

## Test File Organization

**Location:**
- Unit tests: Co-located in the same package directory as source code (e.g. `apps/api/internal/money/money_test.go`, `apps/bot/internal/client/api_client_test.go`).
- Integration tests: Grouped in dedicated integration package `apps/api/tests/integration/` to isolate external dependencies (Docker/PostgreSQL).

**Naming:**
- Files named `<subject>_test.go`
- Test functions named `Test<Subject>_<Scenario>` or `Test<Operation><ExpectedOutcome>` (e.g. `TestNoFlexibleSpendingSoFarGivesFullDailyAllowance`, `TestCalculateNewAveragePriceWeightedRounding`)

**Structure:**

```
apps/api/
├── internal/
│   ├── dateinterval/date_interval_test.go   # Pure algorithmic tests
│   ├── investment/investment_test.go        # Pure logic & average price tests
│   ├── money/money_test.go                  # Precision & rounding tests
│   └── transaction/engine/
│       ├── predictive_engine_test.go        # S2S algorithm tests
│       └── what_if_simulator_test.go        # 12-cycle simulation tests
└── tests/
    └── integration/
        ├── test_support.go                  # Testcontainers PostgreSQL 18 setup
        ├── auth_test.go                     # Real DB Telegram auth tests
        ├── bank_account_test.go             # Real DB bank account tests
        ├── category_test.go                 # Real DB category uniqueness tests
        ├── internal_bot_api_test.go         # Real DB botapi endpoints tests
        ├── investment_test.go               # Real DB investment portfolio tests
        └── transaction_test.go              # Real DB transactions & bundle tests
```

## Test Structure

**Suite Organization:**

```go
func TestCalculateNewAveragePriceIncrementalPurchase(t *testing.T) {
    initialPrice, _ := money.FromFloat(10.00)
    currentLot := Lot{
        Quantity:     decimal.NewFromInt(10),
        AveragePrice: initialPrice,
    }

    newPrice, _ := money.FromFloat(20.00)
    newLot := Lot{
        Quantity:     decimal.NewFromInt(10),
        AveragePrice: newPrice,
    }

    result, err := CalculateNewAveragePrice(currentLot, newLot)
    require.NoError(t, err)
    assert.Equal(t, "15.00", result.String())
}
```

## Mocking

**Framework:**
- `net/http/httptest` for HTTP server mocks in client tests
- No generic mock frameworks (e.g. `gomock`, `testify/mock`) used for database interactions.

**What to Mock:**
- External third-party HTTP services when testing HTTP clients (e.g., in `apps/bot/internal/client/api_client_test.go` using `httptest.NewServer`).

**What NOT to Mock:**
- **DATABASE MOCKS ARE FORBIDDEN:** Under no circumstances use `sqlmock` or in-memory stub drivers. All repository and API endpoint tests execute against real PostgreSQL 18 via Testcontainers (`postgres:18-alpine`).
- Pure math domain objects (`money.Money`, `dateinterval.DateInterval`) are tested directly with real instances.

## Fixtures and Factories

**Test Support:**
- `apps/api/tests/integration/test_support.go` spins up a PostgreSQL 18 container once per test session, runs Goose migrations automatically, and provides isolated database connections:

```go
func SetupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
    // Starts postgres:18-alpine via Testcontainers Go
    // Runs Goose embedded migrations
    // Returns connection pool and cleanup teardown function
}
```

## Coverage

**Requirements:**
- 100% coverage target for pure financial engine and monetary logic (`apps/api/internal/money`, `apps/api/internal/dateinterval`, `apps/api/internal/transaction/engine`).
- Full endpoint coverage via integration tests.

**View Coverage:**

```bash
cd apps/api && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Test Types

**Unit Tests:**
- Scope: Pure algorithms, mathematical edge cases, Banker's rounding, date intervals, financial simulations.
- Speed: Milliseconds, zero I/O, no network or database requirements.

**Integration Tests:**
- Scope: Full HTTP request-response cycle, database constraints, transactional concurrency (`pg_advisory_xact_lock`), foreign key cascades.
- Infrastructure: Automated Docker container via Testcontainers.

**E2E Tests:**
- Executed via integration tests validating complete flows (e.g. create challenge -> authorize challenge -> poll -> retrieve session cookie).

## Common Patterns

**Table-Driven Tests:**

```go
tests := []struct {
    name     string
    amount   string
    parts    int
    expected []string
}{
    {"even split", "100.00", 2, []string{"50.00", "50.00"}},
    {"odd penny split", "10.00", 3, []string{"3.34", "3.33", "3.33"}},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // execute and assert
    })
}
```

---

*Testing analysis: 2026-10-03*
