# Requirements: Meu Dinheiro

**Defined:** 2026-10-03
**Core Value:** Previsibilidade financeira futura sem ansiedade: o usuário sabe exatamente quanto pode gastar hoje (`R$/dia`) sem comprometer contas essenciais ou metas de reserva, e pode simular o impacto de qualquer compra antes de passar o cartão.

## v1 Requirements (MVP - Core Loop `v0.0.1`)

Requisitos prioritários para entrega do loop principal de valor no Telegram (Onboarding Conversacional + Operação Diária com Motor Preditivo).

### Onboarding Conversacional no Telegram (Milestone 2)

- [ ] **ONBD-01**: Usuário interage com uma máquina de estados finitos (State Machine) que conduz o diálogo de boas-vindas e setup passo a passo baseado no seu `telegram_id`
- [ ] **ONBD-02**: Usuário informa seu saldo total inicial de liquidez em contas correntes com validação numérica
- [ ] **ONBD-03**: Usuário define o dia de início do seu ciclo mensal (data de pagamento do salário, de 1 a 28)
- [ ] **ONBD-04**: Usuário informa suas estimativas de despesas fixas essenciais mensais (ex.: aluguel, alimentação, saúde)
- [ ] **ONBD-05**: Usuário recebe cálculo imediato de custos essenciais e escolhe sua meta de Reserva de Emergência via botões inline (6 meses CLT vs 12 meses PJ), definindo seu aporte mensal
- [ ] **ONBD-06**: Usuário pode cadastrar opcionalmente seus ativos e investimentos iniciais (ticker, quantidade e preço)
- [ ] **ONBD-07**: Bot persiste o setup via `POST /internal/users/onboarding` e apresenta o primeiro relatório financeiro com o Saldo Seguro Diário (S2S) e indicador de saúde do ciclo

### Operação Diária & Motor Preditivo no Telegram (Milestone 3)

- [ ] **PRED-01**: Usuário executa o comando `/s2s` e recebe em tempo real o valor disponível para gastar hoje, dias restantes no ciclo e badge de saúde (`SAUDÁVEL`, `RESTRITO`, `RISCO DE DÉFICIT`)
- [ ] **PRED-02**: Usuário registra despesa rápida via `/gasto <valor> <descrição>` (ex.: `/gasto 34.90 Almoço`), gravando a transação via `POST /internal/transactions/quick-expense` e recebendo o S2S recalculado na hora
- [ ] **PRED-03**: Usuário simula compras futuras via `/simular <valor> [parcelas]` (ex.: `/simular 2400 12`), visualizando a projeção de impacto no S2S e alertas de risco de déficit ao longo dos próximos 12 ciclos
- [ ] **PRED-04**: Usuário executa `/checkin` e realiza a conciliação diária guiada de saldo, gravando o snapshot diário em `tb_check_in_snapshots`

---

## v2 Requirements (Investimentos, Proatividade & Web)

Funcionalidades planejadas nos marcos subsequentes do roadmap (`v0.1.0`, `v0.2.0`, `v1.0.0`).

### Carteira de Ações e Investimentos no Telegram (`v0.1.0` - Milestone 4)

- [x] **INVEST-01**: Usuário adiciona ativos à carteira via `/investimento <TICKER>, <QUANTIDADE> un a <PRECO>`, recalculando o Preço Médio ponderado
- [x] **INVEST-02**: Usuário consulta a carteira consolidada via `/investimentos` ou `/carteira`, exibindo ativos, quantidades, preço médio e patrimônio investido
- [x] **INVEST-03**: Usuário realiza venda ou redução de posição via `/venda <TICKER> <QUANTIDADE> a <PRECO>`

### Notificações Proativas & Alertas de Risco (`v0.2.0` - Milestone 5)

- [ ] **NOTIF-01**: Worker envia notificação matinal de "Bom Dia" às 08:00 com o S2S do dia e dias restantes
- [ ] **NOTIF-02**: Sistema dispara alerta proativo imediato se um gasto registrado degradar a saúde do ciclo para `RESTRICTED` ou `DEFICIT_RISK`
- [ ] **NOTIF-03**: Lembrete noturno amigável às 21:00 convidando para fechar o dia com o `/checkin`

### Dashboard Web Completo & Analytics (`v1.0.0` - Milestone 6)

- [ ] **WEB-01**: Usuário visualiza gráfico interativo da curva de liquidez e parcelamentos nos 12 ciclos futuros no frontend Vue 3
- [ ] **WEB-02**: Usuário acessa painel gráfico de investimentos com alocação percentual por ativo
- [ ] **WEB-03**: Usuário gerencia contas bancárias e visualiza extrato analítico completo de transações e bundles parcelados

---

## Out of Scope

| Funcionalidade / Padrão | Razão do Escopo Fora |
|-------------------------|----------------------|
| Autenticação legada por email/senha/BCrypt | Substituída integralmente pela arquitetura nativa Telegram-first |
| Mocks de banco de dados (`sqlmock`) | Proibidos pela diretriz de engenharia; a suíte usa PostgreSQL 18 real via Testcontainers |
| Tipos de ponto flutuante (`float64`) para moedas | Proibido para cálculos financeiros; usa-se sempre `money.Money` com Banker's rounding HalfEven |
| Conexão automática Open Finance bancária | Escopo futuro; no MVP o usuário tem controle total e privacidade através de lançamentos diretos |
| Aplicativo nativo iOS/Android em lojas | O Telegram Bot serve como cliente móvel nativo de altíssima conveniência e sem atrito |

---

## Traceability

Mapeamento de cobertura entre requisitos e as fases do Roadmap.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ONBD-01 | Phase 1 | Complete |
| ONBD-02 | Phase 1 | Complete |
| ONBD-03 | Phase 1 | Complete |
| ONBD-04 | Phase 1 | Complete |
| ONBD-05 | Phase 1 | Complete |
| ONBD-06 | Phase 1 | Complete |
| ONBD-07 | Phase 1 | Complete |
| PRED-01 | Phase 2 | Complete |
| PRED-02 | Phase 2 | Complete |
| PRED-03 | Phase 2 | Complete |
| PRED-04 | Phase 2 | Complete |
| INVEST-01 | Phase 3 | Complete |
| INVEST-02 | Phase 3 | Complete |
| INVEST-03 | Phase 3 | Complete |
| NOTIF-01 | Phase 4 | Pending |
| NOTIF-02 | Phase 4 | Pending |
| NOTIF-03 | Phase 4 | Pending |
| WEB-01 | Phase 5 | Pending |
| WEB-02 | Phase 5 | Pending |
| WEB-03 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-10-03*
*Last updated: 2026-10-03 after initial definition*
