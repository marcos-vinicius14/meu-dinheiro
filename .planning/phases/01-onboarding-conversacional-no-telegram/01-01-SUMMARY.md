---
phase: 01-onboarding-conversacional-no-telegram
plan: "01-01"
subsystem: bot
tags: [telegram, bot, fsm, state-machine, concurrency, session-store, mutex]

# Dependency graph
requires:
  - phase: 01-onboarding-conversacional-no-telegram
    provides: context and architecture decisions
provides:
  - FSM State Machine engine for Telegram bot onboarding
  - In-memory thread-safe SessionStore with RWMutex and TTL eviction worker
  - Update router handling messages and CallbackQuery with /cancelar support
affects: [01-02]

actuals:
  tokens: 4500
  tasks: 2
  commits: 0

tech-stack:
  added: []
  patterns: [State Machine FSM, in-memory session cache with TTL eviction, thread-safe RWMutex store]

key-files:
  created:
    - apps/bot/internal/bot/fsm/types.go
    - apps/bot/internal/bot/fsm/store.go
    - apps/bot/internal/bot/fsm/store_test.go
    - apps/bot/internal/bot/fsm/fsm.go
    - apps/bot/internal/bot/fsm/fsm_test.go
  modified:
    - apps/bot/internal/bot/bot.go

key-decisions:
  - "D-01: SessionStore implementado em memória com sync.RWMutex indexado por telegram_id/chat_id"
  - "D-02: Evicção ativa por TTL (30 min) com ticker em background a cada 5 min"
  - "D-03: Comando /cancelar remove imediatamente a sessão da memória e emite feedback em pt-BR"

patterns-established:
  - "TelegramSender interface: desacoplamento de envio de mensagens e callbacks para testes unitários"
  - "FSM StepHandler pattern: separação limpa entre o despachante de updates e os handlers de diálogo"

requirements-completed: [ONBD-01]

coverage:
  - id: D1
    description: "SessionStore concorrente em memória com sync.RWMutex e evicção por TTL"
    requirement: ONBD-01
    verification:
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/store_test.go#TestSessionStore_ConcurrentAccess"
        status: pass
    human_judgment: false
  - id: D2
    description: "FSM engine com roteamento de mensagens, callbacks e comando /cancelar"
    requirement: ONBD-01
    verification:
      - kind: unit
        ref: "apps/bot/internal/bot/fsm/fsm_test.go#TestFSM_HandleCancelar"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-03
status: complete
---

# Phase 1: Plan 01-01 Summary

**Máquina de Estados Finitos (FSM) em memória com sync.RWMutex, evicção de cache por TTL via worker em background e roteador de mensagens e CallbackQuery com cancelamento defensivo.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-10-03T11:03:00Z
- **Completed:** 2026-10-03T11:06:40Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Implementação dos tipos da FSM e structs de sessão em `apps/bot/internal/bot/fsm/types.go`.
- Implementação do `SessionStore` thread-safe com `sync.RWMutex`, operações O(1) e worker de expurgo ativo a cada 5 min com TTL de 30 min em `store.go`.
- Suíte completa de testes de concorrência (`TestSessionStore_ConcurrentAccess`) rodando sob `-race` com 50 goroutines e zero deadlocks.
- Motor da FSM (`fsm.go`) suportando mensagens de texto, confirmação imediata de `CallbackQuery` e comando `/cancelar` a qualquer momento.
- Integração da FSM e worker de evicção no `Bot` (`apps/bot/internal/bot/bot.go`) preservando o fluxo de login web `/start auth_<token>` e comandos padrão (`/ajuda`, `/status`).

## Files Created/Modified
- `apps/bot/internal/bot/fsm/types.go` - Definições de estados e estruturas de sessão
- `apps/bot/internal/bot/fsm/store.go` - SessionStore com locking concorrente e evicção por TTL
- `apps/bot/internal/bot/fsm/store_test.go` - Testes unitários e de estresse concorrente
- `apps/bot/internal/bot/fsm/fsm.go` - Motor da FSM e despachante de updates Telegram
- `apps/bot/internal/bot/fsm/fsm_test.go` - Testes unitários com mock de TelegramSender
- `apps/bot/internal/bot/bot.go` - Integração da FSM e do SessionStore no bot principal

## Decisions Made
- O método `Set` preserva o `LastActiveAt` caso já esteja definido pelo chamador (útil para testes de expiração) e atualiza para UTC quando zerado.
- `HandleUpdate` retorna `handled = false` para `/start auth_<token>` para garantir que a autorização de login no navegador continue funcionando pelo handler original.

## Deviations from Plan
- None - plan executed exactly as written.

## Issues Encountered
- `TestSessionStore_EvictExpired` falhou na primeira tentativa porque `Set` sobrescrevia incondicionalmente `LastActiveAt` com `now`. Corrigido para preservar timestamps já configurados.

## Next Phase Readiness
- A base da FSM e o SessionStore estão prontos e validados para receber os handlers das etapas do diálogo conversacional no Plano 01-02.

---
*Phase: 01-onboarding-conversacional-no-telegram*
*Completed: 2026-10-03*
