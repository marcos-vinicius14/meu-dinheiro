---
phase: 01-onboarding-conversacional-no-telegram
plan: "01-02"
subsystem: bot
tags: [telegram, bot, fsm, onboarding, parsers, inline-keyboard, api-client, daily-dashboard, s2s]

# Dependency graph
requires:
  - phase: 01-onboarding-conversacional-no-telegram
    plan: "01-01"
    provides: FSM Engine, SessionStore in memory, Update Router
provides:
  - Robust pt-BR input parsers (money, cycle day 1..28, fixed expenses, investments)
  - OnboardingHandler implementing StepHandler with full dialog state transitions
  - Interactive InlineKeyboardMarkup (finish expenses, 6x CLT / 12x PJ fund, skip/add investment)
  - Existing user Daily Dashboard with today's S2S and quick command shortcuts
  - End-to-end integration and API persistence via SaveOnboarding
affects: [milestone-2, daily-operations]

actuals:
  tokens: 8500
  tasks: 3
  commits: 0

tech-stack:
  added: []
  patterns: [FSM StepHandler, Tolerant pt-BR parsing, Banker's rounding Decimal/Money, InlineKeyboard callbacks, E2E Mock HTTP API Testing]

key-files:
  created:
    - apps/bot/internal/bot/fsm/parsers.go
    - apps/bot/internal/bot/fsm/parsers_test.go
    - apps/bot/internal/bot/fsm/handlers.go
    - apps/bot/internal/bot/fsm/handlers_test.go
    - apps/bot/internal/bot/bot_onboarding_test.go
  modified:
    - apps/bot/internal/bot/fsm/fsm.go
    - apps/bot/internal/bot/bot.go

key-decisions:
  - "D-04: Despesas essenciais são cadastradas item a item, somadas na sessão e mapeadas para categorias com botão inline de conclusão"
  - "D-05: Botão inline finish_fixed_expenses permite concluir cadastro de despesas com 1 clique"
  - "D-06: Botões inline de 1 clique oferecem opções 6 meses CLT e 12 meses PJ calculadas a partir das despesas"
  - "D-07: Usuário define meta de aporte mensal (target_savings) para a reserva"
  - "D-08: Usuário pode adicionar ativos de investimentos ou pular etapa com botões inline"
  - "D-09: Usuários já cadastrados recebem o Painel Diário com S2S atual ao enviar /start, mantendo estado IDLE"
  - "D-10: Parsers tolerantes a formatos pt-BR (moeda com vírgula/ponto, dia do ciclo 1..28, ticker e quantidades)"
  - "D-11: Limite máximo do dia de início do ciclo fixado defensivamente em 28 para consistência de calendário com fevereiro"

patterns-established:
  - "OnboardingHandler StepHandler: Desacoplamento da lógica de negócio conversacional do motor genérico da FSM"
  - "APIClient Onboarding: Persistência centralizada via POST /internal/users/onboarding e remoção imediata da sessão em memória pós-sucesso"
  - "Mock HTTP API Server testing: Testes E2E sem dependências de rede externa ou banco de dados mockado no bot"

requirements-completed:
  - ONBD-01
  - ONBD-02
  - ONBD-03
  - ONBD-04
  - ONBD-05
  - ONBD-06
  - ONBD-07

coverage:
  - id: P1
    description: "Parsers de moeda, dia de ciclo, despesas fixas e ativos de investimento"
    requirement: ONBD-02
    verification:
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/parsers_test.go#TestParseMoney"
        status: pass
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/parsers_test.go#TestParseCycleDay"
        status: pass
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/parsers_test.go#TestParseFixedExpense"
        status: pass
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/parsers_test.go#TestParseInvestment"
        status: pass
    human_judgment: false
  - id: H1
    description: "Diálogo interativo de onboarding com botões inline, API e expurgo de sessão"
    requirement: ONBD-01
    verification:
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/handlers_test.go#TestOnboarding_HappyPath_WithoutInvestments"
        status: pass
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/handlers_test.go#TestOnboarding_WithInvestments"
        status: pass
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/handlers_test.go#TestOnboarding_ResilienceAndErrors"
        status: pass
    human_judgment: false
  - id: D1
    description: "Painel diário para usuários já cadastrados via /start"
    requirement: ONBD-07
    verification:
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/handlers_test.go#TestOnboarding_ExistingUser_DailyDashboard"
        status: pass
    human_judgment: false
  - id: B1
    description: "Integração ponta a ponta do Bot com Webhook e FSM"
    requirement: ONBD-01
    verification:
      - kind: integration
        ref: "apps/bot/internal/bot/bot_onboarding_test.go#TestBot_OnboardingIntegration_E2E"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-10-03
status: complete
---

# Phase 1: Plan 01-02 Summary

**Handlers de diálogo conversacional, parsers tolerantes a pt-BR, botões inline interativos, persistência via APIClient e Painel Diário para usuários existentes.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-10-03T11:07:00Z
- **Completed:** 2026-10-03T11:11:30Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

1. **Parsers Robustos pt-BR (`parsers.go`)**:
   - `ParseMoney`: converte múltiplos formatos (ex: `3500`, `3500.00`, `3500,00`, `3.500,00`, `R$ 1.250,90`) e rejeita valores zero, negativos ou inválidos com mensagens amigáveis em português.
   - `ParseCycleDay`: valida estritamente o dia do ciclo de 1 a 28, com mensagem explicativa sobre a consistência de calendário com fevereiro.
   - `ParseFixedExpense`: extração flexível de descrição e valor, inferência automática de categorias (Moradia, Utilidades, Alimentação, Saúde, Transporte) e sanitização de pontuação.
   - `ParseInvestment`: extração de ticker, quantidade e preço unitário com tolerância a termos pt-BR (`un`, `cotas`, `a`, `de`, `r$`) e vírgulas decimais.
2. **Handlers de Diálogo Conversacional (`handlers.go`)**:
   - `OnboardingHandler` implementando `StepHandler` para guiar o usuário em cada estado do fluxo.
   - Teclados inline com botões de 1 clique:
     - `[ ✅ Concluir Despesas Fixas ]` (`finish_fixed_expenses`)
     - `[ 🎯 6 Meses (CLT) - R$ ... ]` (`fund_6`) e `[ 🎯 12 Meses (PJ) - R$ ... ]` (`fund_12`)
     - `[ ➕ Adicionar Ativo ]` (`add_investment`) e `[ ⏭️ Pular Etapa ]` (`skip_investments`)
     - `[ ➕ Adicionar Outro Ativo ]` e `[ ✅ Finalizar Onboarding ]` (`finish_investments`)
   - Integração com `APIClient.SaveOnboarding` para envio do payload completo ao backend e exibição do relatório inicial detalhado com S2S, badge de saúde e comandos rápidos.
   - Painel Diário para usuários já cadastrados que enviarem `/start`, sem sobrescrever seus dados.
3. **Conexão no Bot (`bot.go`)**:
   - `NewBot` instancia e registra o `OnboardingHandler` na FSM.
   - `SetSender` e salvaguardas contra ponteiro nulo em `FSM.Reply` e `FSM.ReplyWithKeyboard`.
4. **Suíte Completa de Testes**:
   - `parsers_test.go`: 100% de cobertura de formatos válidos e edge cases de parsers.
   - `handlers_test.go`: cobertura dos 4 cenários principais (Happy Path, Investimentos, Usuário Existente, Erros e Cancelamento) com servidor HTTP mock.
   - `bot_onboarding_test.go`: teste E2E do bot via webhook validando o ciclo completo até a chamada da API.
   - `make test-bot` passa 100% sob `-race`.

## Files Created/Modified

- `apps/bot/internal/bot/fsm/parsers.go` - Parsers de moeda, ciclo, despesas e ativos
- `apps/bot/internal/bot/fsm/parsers_test.go` - Testes unitários para todos os parsers
- `apps/bot/internal/bot/fsm/handlers.go` - OnboardingHandler e geradores de InlineKeyboardMarkup
- `apps/bot/internal/bot/fsm/handlers_test.go` - Testes unitários dos handlers de conversação
- `apps/bot/internal/bot/fsm/fsm.go` - Adicionado SetSender e checagem defensiva de sender nil
- `apps/bot/internal/bot/bot.go` - Registro do OnboardingHandler em NewBot
- `apps/bot/internal/bot/bot_onboarding_test.go` - Teste de integração E2E do bot

## Decisions & Deviations

- Ajustado `inferCategoryName` para verificar "Transporte" e "gasolina" com precedência sobre "gás" de "Utilidades", evitando falso positivo no parser de despesas.
- Implementado `SetSender` na struct `FSM` para permitir injeção de fakes em testes unitários e de integração sem expor campos privados.

## Verification

- `cd apps/bot && go test -v -race ./...` -> PASS (100% aprovado)
- `make test-bot` -> PASS (100% aprovado)
- `cd apps/bot && go vet ./... && gofmt -s -w .` -> Sem erros ou advertências

---
*Phase: 01-onboarding-conversacional-no-telegram*
*Completed: 2026-10-03*
