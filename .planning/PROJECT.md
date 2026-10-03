# Meu Dinheiro

## What This Is

O **Meu Dinheiro** é um sistema de inteligência financeira pessoal orientado a **previsibilidade futura**. Em vez de focar no registro passivo de gastos passados, o produto calcula o **Saldo Seguro Diário (S2S)** — o valor exato que o usuário pode gastar a cada dia sem comprometer contas fixas ou metas de poupança — e permite simulações de compras parceladas em até 12 ciclos futuros (*what-if*), além de acompanhamento de patrimônio e investimentos. A interface primária de operação diária é o **Bot do Telegram**, com uma aplicação Web em Vue 3 para autenticação 1-click e visualizações analíticas de longo prazo.

## Core Value

Previsibilidade financeira sem ansiedade: o usuário sabe exatamente quanto pode gastar hoje (`R$/dia`) sem comprometer contas essenciais ou metas de reserva, e pode simular o impacto de qualquer compra antes de passar o cartão.

## Requirements

### Validated

- ✓ Autenticação Telegram-first sem senhas com desafio criptográfico efêmero, deep link e sessão via cookies JWT HTTP-only — existing
- ✓ Motor matemático de cálculo determinístico de Saldo Seguro Diário (S2S) com Banker's rounding HalfEven (`money.Money`) — existing
- ✓ Simulador preditivo what-if multi-ciclo de 1 a 12 meses futuros com detecção de gargalos e riscos de déficit — existing
- ✓ Gestão de contas bancárias com locking transacional via `pg_advisory_xact_lock` e teto defensivo de 3 contas — existing
- ✓ Gestão de categorias com unicidade case-insensitive por usuário — existing
- ✓ Gestão de transações avulsas, pacotes parcelados imutáveis e snapshots de check-in diário — existing
- ✓ Provisionamento oficial do bot no Telegram (`Julius` / `@meudinheiro_app_bot`) com menu de comandos registrado no BotFather — existing (M0)
- ✓ Suporte dual a Long Polling (desenvolvimento) e Webhook HTTPS seguro com validação de secret token — existing (M0)
- ✓ Esquema de banco de dados para investimentos (`tb_investments`) e perfil financeiro na `tb_users` no PostgreSQL 18 — existing (M1)
- ✓ Cálculo de ciclo financeiro dinâmico (`dateinterval.CycleOf`, dias 1 a 28) adaptado à data de recebimento do usuário — existing (M1)
- ✓ Módulo `internal/investment` com cálculo de Preço Médio ponderado, acréscimo de lotes e abate de posições — existing (M1)
- ✓ Cálculo de custos fixos mensais essenciais e recomendação inteligente de Reserva de Emergência (6x CLT vs 12x PJ) — existing (M1)
- ✓ Endpoints internos de comunicação Bot ↔ API (`internal/botapi`) e cliente Go no bot (`apps/bot/internal/client`) — existing (M1)

### Active

- [ ] **M2-ONBOARD-01**: Máquina de estados conversacional (State Machine) no Telegram para guiar o diálogo de configuração inicial do usuário
- [ ] **M2-ONBOARD-02**: Coleta interativa de saldo inicial somando contas correntes via chat
- [ ] **M2-ONBOARD-03**: Configuração interativa do dia de início do ciclo mensal (dia do salário, entre 1 e 28)
- [ ] **M2-ONBOARD-04**: Estimativa e registro de despesas fixas essenciais (moradia, alimentação, saúde)
- [ ] **M2-ONBOARD-05**: Recomendação interativa de Reserva de Emergência com botões inline de 1 clique (6x CLT vs 12x PJ) e definição do aporte mensal
- [ ] **M2-ONBOARD-06**: Cadastro de posições iniciais de investimentos (ticker, quantidade, preço) via diálogo
- [ ] **M2-ONBOARD-07**: Conclusão do onboarding via `POST /internal/users/onboarding` e entrega do primeiro relatório com S2S diário e saúde do ciclo
- [ ] **M3-DAILY-01**: Comando `/s2s` no Telegram para consulta em tempo real do Saldo Seguro Diário, dias restantes e indicador de saúde do ciclo
- [ ] **M3-DAILY-02**: Comando `/gasto <valor> <descrição>` no Telegram para registro imediato de despesa flexível com recalibração dinâmica do S2S
- [ ] **M3-DAILY-03**: Comando `/simular <valor> [parcelas]` no Telegram executando o simulador what-if de até 12 ciclos futuros com alertas de risco
- [ ] **M3-DAILY-04**: Comando `/checkin` no Telegram para wizard interativo de conciliação diária de saldo e registro de snapshots

### Out of Scope

- Autenticação legada por email/senha/BCrypt — substituída integralmente pelo modelo Telegram-first
- Mocks de banco de dados (`sqlmock`) — proibidos pela diretriz de engenharia; testes usam PostgreSQL 18 real com Testcontainers
- Uso de tipos de ponto flutuante (`float32`/`float64`) para valores monetários — proibido; utiliza-se sempre `money.Money` com precisão decimal
- Integração bancária Open Finance automatizada no MVP — priorizada a entrada conversacional rápida e sob controle do usuário
- Aplicativo mobile nativo (iOS/Android) — o Telegram Bot atua como cliente móvel nativo de alta conveniência

## Context

- **Monorepo:** Go (`apps/api` e `apps/bot`), Vue 3 + Vite (`apps/web`), PostgreSQL 18 em Docker (`compose.yaml`).
- **Arquitetura:** Monólito modular em Go com separação estrita de domínios, sem ORM (queries SQL diretas com `jackc/pgx/v5`).
- **Estado Atual da Implementação:**
  - Milestones 0 e 1 estão implementados e validados no backend e nos contratos do cliente do bot.
  - A worktree contém o código dos endpoints internos e módulos de investimentos prontos para uso.
  - O próximo passo imediato é implementar o fluxo de Onboarding Conversacional (Milestone 2) e os Comandos de Operação Diária (Milestone 3) no `apps/bot`.

## Constraints

- **Linguagem & Idioma**: Mensagens de erro e respostas para o usuário devem ser em Português (pt-BR). Identificadores de código em inglês.
- **Precisão Financeira**: Valores monetários obrigatoriamente manipulados via `decimal.Decimal` / `money.Money` com arredondamento `RoundBank` (HalfEven).
- **Concorrência**: Mutações em recursos compartilhados e com limites defensivos usam `pg_advisory_xact_lock` no PostgreSQL 18.
- **Segurança**: Endpoints internos da API protegidos por cabeçalho secreto `X-Internal-Secret` compartilhado com o bot.
- **Git**: É expressamente proibido realizar `git commit` ou `git push` sem autorização formal e prévia do usuário.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Telegram-First como interface primária | O usuário toma decisões de compra no ponto de venda; o Telegram oferece fricção zero sem senhas | ✓ Good |
| Mapeamento de moedas com `decimal.Decimal` | Evita aberrações de arredondamento inerentes a float em cálculos diários de S2S | ✓ Good |
| Testes com Testcontainers e PostgreSQL 18 real | Elimina falsos positivos de mocks e garante paridade absoluta com produção | ✓ Good |
| Máquina de Estados no Bot para Onboarding | Garante que o usuário complete a configuração mínima indispensável para o cálculo do S2S | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-10-03 after initialization*
