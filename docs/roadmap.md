# Roadmap do Produto — Meu Dinheiro

> **Visão do Produto:** Sistema de inteligência financeira pessoal focado em **previsibilidade futura**. O diferencial não é olhar para o passado ou categorizar gastos passivos, mas sim projetar o fluxo futuro através do **Saldo Seguro Diário (S2S)**, simulações de impacto de compras em até 12 ciclos (*what-if*) e acompanhamento de patrimônio e investimentos.
>
> **Estratégia de Lançamento:** O **Bot do Telegram** é a interface primária e prioritária de operação diária (onde o usuário vive e toma decisões de compra). O **Dashboard Web** virá em seguida para relatórios aprofundados e visualizações gráficas de longo prazo.
>
> **Estratégia de Releases:**
> - 🚀 **`v0.0.1` (MVP - Core Loop)**: M1 (APIs Bot) + M2 (Onboarding) + M3 (Operação Diária: `/s2s`, `/gasto`, `/simular`).
> - 📈 **`v0.1.0` (Investimentos)**: M4 (Carteira de Ações & Preço Médio).
> - ⏰ **`v0.2.0` (Proatividade)**: M5 (Workers Agendados & Alertas de Risco).
> - 🌐 **`v1.0.0` (Web & Analytics)**: M6 (Dashboard Web Completo).

---

## 🗺️ Visão Geral dos Milestones

```mermaid
graph TD
    M0["Milestone 0: Criação & Configuração do Bot (✅ Concluído)"] --> M1
    subgraph V001 ["🚀 Versão 0.0.1 (MVP - Core Loop)"]
        M1["Milestone 1: Backend de Investimentos & APIs do Bot (API)"] --> M2["Milestone 2: Onboarding Conversacional no Telegram (Bot)"]
        M2 --> M3["Milestone 3: Comandos do Motor Preditivo & S2S no Chat (Bot)"]
    end
    M3 --> M4["Milestone 4: Gestão de Ativos e Carteira de Ações (Bot) — v0.1.0"]
    M4 --> M5["Milestone 5: Notificações Proativas & Alertas de Risco (Bot/Worker) — v0.2.0"]
    M5 --> M6["Milestone 6: Dashboard Web Completo & Gráficos 12 Ciclos (Web) — v1.0.0"]
```

---

## 📍 Detalhamento dos Milestones

### ✅ Milestone 0: Criação & Configuração Oficial do Bot no Telegram
**Foco:** Provisionamento do Bot no ecossistema do Telegram, definição de credenciais, comandos nativos e arquitetura de recepção de mensagens.

- [x] **Provisionamento no @BotFather**:
  - [x] Criar o bot oficial via comando `/newbot` no `@BotFather` (`Julius` / `@meudinheiro_app_bot`).
  - [x] Definir descrição do bot (`/setdescription`) e bio (`/setabouttext`).
  - [x] Obter o token de API (`TELEGRAM_BOT_TOKEN`) e adicioná-lo com segurança ao `.env`.
- [x] **Configuração de Comandos no Telegram (`/setcommands`)**:
  - [x] Registrar a lista de comandos no menu nativo do Telegram para auto-completar:
    ```text
    s2s - Consultar seu Saldo Seguro Diário e saúde do ciclo
    gasto - Registrar uma despesa rápida (ex: /gasto 45 Almoço)
    simular - Projetar compra parcelada em até 12 ciclos futuros
    investimento - Adicionar ativo à carteira (ex: /investimento ALUP11, 10 un a 42.23)
    investimentos - Visualizar patrimônio e carteira de ações
    checkin - Fazer o check-in diário e conciliação de saldo
    ajuda - Instruções e lista de comandos
    ```
- [x] **Arquitetura de Comunicação (Long Polling vs. Webhook)**:
  - [x] Suporte nativo a **Long Polling** para desenvolvimento local sem necessidade de túnel ou IP público.
  - [x] Suporte a **Webhook** HTTPS com validação estrita de secret token (`X-Telegram-Bot-Api-Secret-Token`) e endpoint de healthcheck (`/health`) para deploy em produção no Coolify.

---

### ✅ Milestone 1: Backend de Investimentos, Gastos Essenciais & APIs para o Bot `[v0.0.1]`
**Foco:** Preparar o modelo de dados e endpoints internos na `apps/api` para suportar patrimônio em ações, ciclo personalizado, recomendação inteligente de reserva e a operação direta do Bot via `telegram_id`.

- [x] **Esquema de Banco de Dados (`tb_investments` e perfil em `tb_users`)**:
  - Nova tabela no PostgreSQL 18:
    - `id UUID PRIMARY KEY DEFAULT uuidv7()`
    - `user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE`
    - `ticker VARCHAR(12) NOT NULL` (ex: `ALUP11`, `BBAS3`, `IVVB11` normalizado em uppercase)
    - `quantity NUMERIC(15, 4) NOT NULL CHECK (quantity > 0)`
    - `average_price NUMERIC(15, 2) NOT NULL CHECK (average_price >= 0)`
    - `created_at` e `updated_at` TIMESTAMPTZ
    - `UNIQUE(user_id, ticker)` com suporte a compras incrementais recalculando o preço médio ponderado.
  - Extensão da `tb_users` com colunas de perfil e metas:
    - `target_savings NUMERIC(19, 2) NOT NULL DEFAULT 0.00`
    - `flexible_budget_cap NUMERIC(19, 2) NOT NULL DEFAULT 0.00`
    - `emergency_fund_target NUMERIC(19, 2) NOT NULL DEFAULT 0.00`
    - `emergency_fund_months INTEGER NOT NULL DEFAULT 6`
    - `cycle_start_day INTEGER NOT NULL DEFAULT 1 CHECK (cycle_start_day BETWEEN 1 AND 28)`
- [x] **Ciclo Financeiro Dinâmico (`dateinterval.CycleOf`)**:
  - Cálculo determinístico de ciclos mensais personalizados iniciando no dia definido pelo usuário (`cycle_start_day`, default 1).
- [x] **Módulo `internal/investment` (Go)**:
  - CRUD de investimentos com arredondamento `decimal.RoundBank` (HalfEven).
  - Cálculo de Preço Médio Ponderado na adição de novos lotes:
    $$\text{Novo PM} = \frac{(\text{Qtd Atual} \times \text{PM Atual}) + (\text{Qtd Nova} \times \text{Preço Novo})}{\text{Qtd Total}}$$
  - Abatimento de posição (vendas) mantendo o PM e deleção automática ao zerar.
  - Endpoints REST autenticados para web (`/investments`).
- [x] **Gastos Essenciais & Recomendação Inteligente de Reserva de Emergência**:
  - Cálculo automático de custo fixo mensal somando despesas essenciais.
  - Projeção de reserva sugerida em 6 meses (CLT) e 12 meses (PJ/autônomo).
  - Métricas em tempo real no contexto: meses cobertos e progresso da meta.
- [x] **Endpoints de Comunicação Bot ↔ API (Pacote `internal/botapi`)**:
  - `POST /internal/users/onboarding`: Salva saldo inicial, ciclo, gastos essenciais, reserva 6x/12x, aporte mensal e investimentos.
  - `GET /internal/users/context-by-telegram?telegram_id=...`: Retorna o contexto financeiro completo (contas, ciclo, S2S atual, reserva e investimentos) protegido por `X-Internal-Secret` ou `X-Internal-API-Key`.
  - `POST /internal/investments`: Registro ou incremento de ativos vinculados ao `telegram_id`.
  - `POST /internal/transactions/quick-expense`: Lançamento direto de despesa via bot com cálculo imediato do impacto no S2S.
- [x] **Cliente Go no Bot (`apps/bot/internal/client`)**:
  - Implementação de `SaveOnboarding`, `GetUserContextByTelegram`, `AddInvestment` e `QuickExpense` com testes unitários.

---

### 🟢 Milestone 2: Onboarding Conversacional no Telegram `[v0.0.1]`
**Foco:** Prover a primeira experiência de uso encantadora e guiada logo após o `/start` ou autorização de login.

- [ ] **Máquina de Estados de Conversação (State Machine)**:
  - Gerenciador de estado de diálogo em memória ou banco para cada `telegram_id`:
    - `STATE_IDLE`
    - `STATE_ONBOARDING_BALANCE`
    - `STATE_ONBOARDING_CYCLE_DAY`
    - `STATE_ONBOARDING_FIXED_EXPENSES`
    - `STATE_ONBOARDING_EMERGENCY_FUND_CHOICE`
    - `STATE_ONBOARDING_SAVINGS_TARGET`
    - `STATE_ONBOARDING_HAS_INVESTMENTS`
    - `STATE_ONBOARDING_INVESTMENT_TICKER`
    - `STATE_ONBOARDING_INVESTMENT_QTY`
    - `STATE_ONBOARDING_INVESTMENT_PRICE`
- [ ] **Fluxo Guiado de Boas-Vindas**:
  1. **Boas-vindas:** Explicação rápida do método de Saldo Seguro Diário e previsibilidade.
  2. **Pergunta 1 (Liquidez):** *"Para começar, qual o seu saldo total somando suas contas correntes hoje? (Ex: 3500.00)"*
  3. **Pergunta 2 (Início do Ciclo):** *"Em qual dia costuma cair seu salário para reiniciarmos seu ciclo mensal? (Padrão: dia 01)"*
  4. **Pergunta 3 (Gastos Essenciais):** *"Quanto você estima gastar por mês com despesas essenciais como moradia/aluguel, mercado e saúde? (Ex: 1500 aluguel, 800 mercado)"*
  5. **Pergunta 4 (Reserva de Emergência):**
     - O bot calcula na hora:
       > *"Seu custo essencial mensal é de R$ 2.300. Para sua segurança, recomendamos montar uma Reserva de Emergência. Você prefere uma meta de **6 meses (R$ 13.800)** ou **12 meses (R$ 27.600)**?"*
     - Botões inline de 1 clique: `[ 6 Meses (R$ 13.8k) ]` ou `[ 12 Meses (R$ 27.6k) ]`.
     - Em seguida pergunta o aporte mensal: *"Quanto deseja guardar por mês para essa reserva? (Ex: 300.00)"*
  6. **Pergunta 5 (Investimentos):** *"Você possui dinheiro investido em ações ou outros ativos? (Sim / Não)"*
     - Se "Sim": cadastra ticker, quantidade e preço (ex: `ALUP11 10 unidades a 42.23`).
  7. **Cálculo Inicial:** O bot roda o motor preditivo e entrega o primeiro relatório:
     > *"✅ Configuração concluída! Seu Saldo Seguro Diário (S2S) para os próximos 30 dias é **R$ 78,50/dia**. Status: **SAUDÁVEL**.\n"*
     > *"🛡️ Sua Reserva de Emergência cobre atualmente **1,5 meses** da sua meta de 6 meses."*

---

### 🟢 Milestone 3: Comandos do Motor Preditivo & Operação Diária `[v0.0.1]`
**Foco:** Integrar todos os superpoderes do motor matemático diretamente no chat do Telegram.

- [ ] **Comando `/s2s`**:
  - Consulta o Saldo Seguro Diário em tempo real.
  - Exibe dias restantes do ciclo, limite flexível diário e badge de saúde financeira (`🟢 SAUDÁVEL`, `🟡 RESTRITO`, `🔴 RISCO DE DÉFICIT`).
- [ ] **Comando `/gasto <valor> <descrição>`**:
  - Exemplo: `/gasto 34.90 Almoço` ou `/gasto 120 Mercado`.
  - Cria transação do tipo `EXPENSE` e retorna o impacto imediato no S2S:
    > *"Gasto de R$ 34,90 registrado em Alimentação. Seu novo S2S para hoje é **R$ 78,12**."*
- [ ] **Comando `/simular <valor> [parcelas]` (Método do Breno)**:
  - Exemplo: `/simular 2400 12` (Compra de R$ 2.400 em 12x).
  - Executa o simulador *what-if* de 1 a 12 ciclos futuros na API.
  - Responde ao usuário com diagnóstico preditivo:
    > *"🔮 **Simulação de Compra: R$ 2.400 em 12x de R$ 200,00**\n\n"*
    > *"⚠️ **Atenção:** Essa compra reduzirá seu S2S de R$ 85/dia para R$ 51/dia e gerará risco de déficit no **Ciclo 5 (Maio)**.\n"*
    > *"Recomendação: Aguarde a liquidação de parcelas anteriores antes de assumir esse novo parcelamento."*
- [ ] **Comando `/checkin`**:
  - Wizard interativo para conciliar o saldo do dia e registrar o snapshot diário em `tb_check_in_snapshots`.

---

### 🟢 Milestone 4: Gestão de Ativos & Carteira de Ações `[v0.1.0]`
**Foco:** Permitir que o usuário acompanhe e expanda sua carteira de investimentos pelo Telegram.

- [ ] **Comando `/investimento <TICKER>, <QUANTIDADE> unidades a <PRECO>`**:
  - Expressão regular flexível para aceitar variações comuns:
    - `/investimento ALUP11, 10 unidades a 42.23`
    - `/investimento ALUP11 10 42.23`
    - `/investimento PETR4 100 cotas a 38.50`
  - Se o ativo já existir na carteira, calcula o novo Preço Médio ponderado e soma a posição.
- [ ] **Comando `/investimentos` ou `/carteira`**:
  - Lista todos os ativos cadastrados, posições e patrimônio investido:
    ```text
    📊 Sua Carteira de Investimentos

    • ALUP11: 10 un. | PM: R$ 42,23 | Total: R$ 422,30
    • BBAS3:  50 un. | PM: R$ 27,50 | Total: R$ 1.375,00
    • IVVB11:  5 un. | PM: R$ 310,00| Total: R$ 1.550,00
    --------------------------------------------------
    Total Investido: R$ 3.347,30
    Patrimônio Total (Contas + Ações): R$ 8.120,50
    ```
- [ ] **Comando `/venda <TICKER> <QUANTIDADE> a <PRECO>`**:
  - Abatimento de posição na carteira e lançamento opcional do crédito na conta bancária.

---

### 🟢 Milestone 5: Notificações Proativas & Alertas de Risco `[v0.2.0]`
**Foco:** O bot deixa de ser apenas reativo e passa a ser um assistente pessoal ativo.

- [ ] **Worker de Bom Dia (S2S Matinal)**:
  - Envio agendado (ex.: 08:00): *"Bom dia! Seu S2S disponível para hoje é R$ 92,00. 18 dias restantes no ciclo."*
- [ ] **Alerta de Degradação de Ciclo**:
  - Se um gasto registrado mudar o status de `HEALTHY` para `RESTRICTED` ou `DEFICIT_RISK`, dispara alerta preventivo imediato.
- [ ] **Lembrete de Check-in Noturno**:
  - Notificação amigável às 21:00 convidando para fechar o dia com `/checkin`.

---

### 🟢 Milestone 6: Dashboard Web Completo & Projeção Gráfica `[v1.0.0]`
**Foco:** Interface visual rica em Vue 3 para planejamento estratégico e relatórios.

- [ ] **Gráfico de Projeção dos 12 Ciclos**:
  - Linha do tempo visual exibindo o fluxo de caixa projetado mês a mês.
  - Curva de saldo livre vs. parcelamentos vigentes vs. reserva acumulada.
- [ ] **Painel de Investimentos**:
  - Gráfico de pizza por ativo/classe e tabela de alocação de patrimônio.
- [ ] **Extrato Analítico & Gestão de Contas**:
  - Filtros avançados, edição e cancelamento de bundles/parcelamentos.

---

## 📋 Resumo das Fases de Entrega

| Versão Alvo | Milestone | Escopo Principal | Entrega | Status |
|---|---|---|---|---|
| **`v0.0.1`** | **M0** | Criação, Token, Comandos e Webhook do Bot | Telegram / `apps/bot` | ✅ Concluído |
| **`v0.0.1`** *(MVP Core Loop)* | **M1** | Backend de Investimentos & APIs Bot | `apps/api` | ✅ Concluído |
| **`v0.0.1`** *(MVP Core Loop)* | **M2** | Onboarding Conversacional & Saldo Inicial | `apps/bot` | ⏳ Planejado |
| **`v0.0.1`** *(MVP Core Loop)* | **M3** | Comandos S2S, /gasto e Simulador What-If | `apps/bot` | ⏳ Planejado |
| **`v0.1.0`** | **M4** | Comando `/investimento` & Carteira de Ações | `apps/bot` | ⏳ Planejado |
| **`v0.2.0`** | **M5** | Notificações Proativas & Worker S2S | `apps/bot` + Worker | ⏳ Planejado |
| **`v1.0.0`** | **M6** | Dashboard Web Completo & Gráficos 12 Meses | `apps/web` | ⏳ Futuro |
