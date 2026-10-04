---
phase: "03"
slug: "gest-o-de-carteira-de-a-es-no-telegram-v0-1-0"
status: complete
nyquist_compliant: true
wave_0_complete: true
created: "2026-10-04"
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` + Testcontainers (PostgreSQL 18) |
| **Config file** | `Makefile`, `apps/api/go.mod`, `apps/bot/go.mod` |
| **Quick run command** | `cd apps/bot && go test -v ./internal/bot/... ./internal/bot/fsm/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd apps/bot && go test -v ./internal/bot/... ./internal/bot/fsm/...` ou `cd apps/api && go test -v ./tests/integration -run 'TestInternalBotAPI_.*Investment'`
- **After every plan wave:** Run `make test-bot` e `make test-api`
- **Before `/gsd-verify-work`:** Full suite must be green (`make test`)
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | INVEST-02 | T-03-01 / — | Endpoint interno protegido por segredo retorna ativos e totais | integration | `cd apps/api && go test -v -tags=integration ./tests/integration -run TestInternalBotAPI_Investments` | ✅ | ✅ green |
| 03-01-02 | 01 | 1 | INVEST-03 | T-03-02 / — | Baixa atômica de custódia com row locking e validação de quantidade | integration | `cd apps/api && go test -v -tags=integration ./tests/integration -run TestInternalBotAPI_Investments` | ✅ | ✅ green |
| 03-01-03 | 01 | 2 | INVEST-01 | T-03-03 / — | Parser aceita tickers flexíveis de ações e renda fixa (até 12 chars) | unit | `cd apps/bot && go test -v ./internal/bot/fsm/... -run TestParseInvestment` | ✅ | ✅ green |
| 03-01-04 | 01 | 2 | INVEST-03 | T-03-04 / — | Parser extrai ticker, quantidade e preço opcional de /venda | unit | `cd apps/bot && go test -v ./internal/bot/fsm/... -run TestParseSaleCommand` | ✅ | ✅ green |
| 03-01-05 | 01 | 3 | INVEST-01 | T-03-05 / — | Comandos /investimento, /aporte e /comprar adicionam lote e recalculam PM | unit | `cd apps/bot && go test -v -race ./internal/bot/... -run TestInvestCommand` | ✅ | ✅ green |
| 03-01-06 | 01 | 3 | INVEST-02 | T-03-06 / — | Comando /carteira exibe cards ordenados por volume R$, % e rodapé | unit | `cd apps/bot && go test -v -race ./internal/bot/... -run TestPortfolioCommand` | ✅ | ✅ green |
| 03-01-07 | 01 | 3 | INVEST-03 | T-03-07 / — | Comando /venda abate custódia, apura P&L e trata callbacks | unit | `cd apps/bot && go test -v -race ./internal/bot/... -run TestSellCommand` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `apps/api/tests/integration/internal_bot_api_investment_test.go` — stubs e testes para `GET /internal/investments` e `POST /internal/investments/sell`
- [x] `apps/bot/internal/bot/fsm/parsers_test.go` — testes de `ParseSaleCommand` e expansão de `tickerRegex`
- [x] `apps/bot/internal/bot/bot_invest_test.go` — testes para comandos de investimentos e callbacks inline

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| *None* | — | — | All phase behaviors have automated verification. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
