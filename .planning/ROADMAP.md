# Roadmap: Meu Dinheiro

## Overview

O Meu Dinheiro transforma a gestão financeira pessoal ao substituir o registro passivo de despesas por previsibilidade futura através do cálculo diário do Saldo Seguro Diário (S2S), simulações what-if de parcelamentos e acompanhamento de carteira. Com a base de dados, regras de negócio e endpoints internos prontos na API Go (`apps/api`), o roadmap prioriza a entrega do loop principal de valor no Bot do Telegram (`v0.0.1`), seguido por gestão de ações (`v0.1.0`), proatividade (`v0.2.0`) e dashboard analítico web (`v1.0.0`).

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3...): Planned milestone work
- Decimal phases (1.1, 2.1...): Urgent insertions (marked with INSERTED)

- [x] **Phase 1: Onboarding Conversacional no Telegram** - Máquina de estados conversacional, coleta de saldo inicial, ciclo, gastos essenciais, reserva 6x/12x e entrega do primeiro S2S.
- [ ] **Phase 2: Comandos do Motor Preditivo & Operação Diária** - Implementação dos comandos `/s2s`, `/gasto`, `/simular` (what-if 12 ciclos) e `/checkin` diário no chat.
- [ ] **Phase 3: Gestão de Carteira de Ações no Telegram (`v0.1.0`)** - Comandos `/investimento`, `/investimentos` e `/venda` com recálculo de preço médio ponderado.
- [ ] **Phase 4: Notificações Proativas & Alertas de Risco (`v0.2.0`)** - Workers agendados de Bom Dia (S2S matinal), alerta de degradação da saúde do ciclo e lembrete de check-in.
- [ ] **Phase 5: Dashboard Web Completo & Gráficos 12 Ciclos (`v1.0.0`)** - Interface analítica rica em Vue 3 com curva de liquidez futura e painel de investimentos.

## Phase Details

### Phase 1: Onboarding Conversacional no Telegram

**Goal**: Conduzir o usuário pelo diálogo interativo de boas-vindas no Telegram para capturar liquidez inicial, dia do ciclo, despesas essenciais, escolha da reserva 6x/12x e investimentos, salvando o setup via API e entregando o primeiro relatório de S2S.
**Mode:** mvp
**Depends on**: Nothing (Backend M1 já concluído e disponível)
**Requirements**: [ONBD-01, ONBD-02, ONBD-03, ONBD-04, ONBD-05, ONBD-06, ONBD-07]
**Success Criteria** (what must be TRUE):
  1. Usuário envia `/start` e é recebido com apresentação do método e início da máquina de estados conversacional.
  2. Bot valida o saldo inicial e dia do ciclo (1 a 28) com tratamento amigável de entradas inválidas em pt-BR.
  3. Bot calcula o custo essencial mensal e apresenta botões inline para escolha da meta de Reserva de Emergência (6 meses CLT vs 12 meses PJ), solicitando o aporte mensal planejado.
  4. Bot permite cadastrar ativos de investimento iniciais de forma opcional.
  5. Ao finalizar, o bot envia o payload completo para `POST /internal/users/onboarding` e exibe o resumo com o Saldo Seguro Diário (S2S) e status de saúde do ciclo.

**Plans**: 2 plans

Plans:
**Wave 1**
- [x] 01-01: Implementar máquina de estados finitos (State Machine) e rotas de mensagens no pacote `apps/bot/internal/bot`

**Wave 2**
- [x] 01-02: Implementar handlers de cada etapa do diálogo, botões inline de reserva, integração com APIClient (`SaveOnboarding`) e testes unitários

---

### Phase 2: Comandos do Motor Preditivo & Operação Diária

**Goal**: Disponibilizar no Telegram os comandos centrais do motor preditivo para consulta instantânea do S2S, registro rápido de despesas com recalibração imediata, simulações what-if de até 12 ciclos e conciliação diária de saldo.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: [PRED-01, PRED-02, PRED-03, PRED-04]
**Success Criteria** (what must be TRUE):
  1. Usuário digita `/s2s` e recebe imediatamente o saldo seguro do dia, dias restantes no ciclo e badge de saúde financeira (`SAUDÁVEL`, `RESTRITO`, `RISCO DE DÉFICIT`).
  2. Usuário digita `/gasto 35 Almoço`, a transação é criada via `POST /internal/transactions/quick-expense` e o bot responde com o novo S2S atualizado.
  3. Usuário digita `/simular 2400 12`, a simulação what-if de 12 ciclos é executada e o bot alerta sobre eventuais impactos ou riscos de déficit futuro.
  4. Usuário executa `/checkin` e realiza a conciliação diária com snapshot salvo na base.

**Plans**: 2 plans

Plans:
- [ ] 02-01: Implementar handlers dos comandos `/s2s` e `/gasto` com parsing de argumentos e feedback de recalibração
- [ ] 02-02: Implementar handlers dos comandos `/simular` (integração what-if) e `/checkin` (conciliação diária) com testes

---

### Phase 3: Gestão de Carteira de Ações no Telegram (`v0.1.0`)

**Goal**: Permitir ao investidor gerenciar sua carteira de ativos diretamente pelo chat, atualizando preços médios ponderados e acompanhando o patrimônio total.
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: [INVEST-01, INVEST-02, INVEST-03]
**Success Criteria** (what must be TRUE):
  1. Usuário digita `/investimento ALUP11, 10 un a 42.23`, adiciona o ativo e recebe o novo Preço Médio ponderado calculado.
  2. Usuário digita `/investimentos` ou `/carteira` e visualiza uma tabela formatada com seus ativos, quantidade, PM e total investido.
  3. Usuário pode abater posições via `/venda`.

**Plans**: 1 plan

Plans:
- [ ] 03-01: Implementar parser flexível de ativos e comandos `/investimento`, `/investimentos` e `/venda` no bot

---

### Phase 4: Notificações Proativas & Alertas de Risco (`v0.2.0`)

**Goal**: Tornar o bot um assistente ativo através de rotinas agendadas de envio matinal de S2S, alertas instantâneos de risco e lembretes noturnos.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: [NOTIF-01, NOTIF-02, NOTIF-03]
**Success Criteria** (what must be TRUE):
  1. Worker matinal envia o S2S diário às 08:00 para todos os usuários cadastrados.
  2. Registro de despesa que altera o status para `RESTRICTED` ou `DEFICIT_RISK` dispara alerta preventivo imediato.
  3. Lembrete noturno às 21:00 convida o usuário para o check-in diário.

**Plans**: 1 plan

Plans:
- [ ] 04-01: Implementar scheduler/cron interno para disparo das notificações proativas e listener de alertas

---

### Phase 5: Dashboard Web Completo & Gráficos 12 Ciclos (`v1.0.0`)

**Goal**: Desenvolver no frontend Vue 3 os componentes visuais avançados para análise estratégica de fluxo de caixa, simulação gráfica e alocação patrimonial.
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: [WEB-01, WEB-02, WEB-03]
**Success Criteria** (what must be TRUE):
  1. Painel web exibe gráfico interativo da curva de liquidez para os próximos 12 ciclos.
  2. Página de investimentos exibe gráficos de distribuição percentual e histórico de compras.
  3. Gerenciamento web de contas bancárias e extrato analítico com cancelamento de parcelas.

**Plans**: 2 plans

Plans:
- [ ] 05-01: Implementar visualizações gráficas de fluxo de caixa de 12 ciclos e carteira de investimentos no Vue 3
- [ ] 05-02: Implementar gestão web de contas bancárias e extrato analítico de transações

---

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Onboarding Conversacional no Telegram | v0.0.1 (MVP) | 2/2 | Complete | 2026-10-03 |
| 2. Comandos do Motor Preditivo & Operação Diária | v0.0.1 (MVP) | 0/2 | Not started | - |
| 3. Gestão de Carteira de Ações no Telegram | v0.1.0 | 0/1 | Not started | - |
| 4. Notificações Proativas & Alertas de Risco | v0.2.0 | 0/1 | Not started | - |
| 5. Dashboard Web Completo & Gráficos 12 Ciclos | v1.0.0 | 0/2 | Not started | - |
