# Phase 1: Onboarding Conversacional no Telegram - Context

**Gathered:** 2026-10-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Entrega do fluxo de onboarding conversacional no bot do Telegram (`apps/bot`), conduzindo o novo usuário desde o primeiro `/start` através de uma máquina de estados finitos (State Machine) interativa. O fluxo coleta saldo inicial, dia de início do ciclo mensal (1 a 28), despesas essenciais detalhadas (criando categorias correspondentes na API), definição da meta de reserva de emergência (6 meses CLT vs 12 meses PJ via botões inline) e respectivo aporte mensal de poupança, e cadastro opcional de posições de investimento com botões inline. Ao final, submete o setup via `APIClient.SaveOnboarding` (`POST /internal/users/onboarding`) e apresenta o primeiro relatório financeiro com o Saldo Seguro Diário (S2S), indicador de saúde do ciclo (`SAUDÁVEL`) e cobertura da reserva. Para usuários já cadastrados que enviarem `/start`, exibe diretamente o painel financeiro diário com o S2S do dia e menu de comandos.

Corresponde integralmente ao **Milestone 2** de `docs/roadmap.md` e aos requisitos `ONBD-01` a `ONBD-07`.

</domain>

<decisions>
## Implementation Decisions

### Máquina de Estados & Persistência de Sessão
- **D-01:** O gerenciamento do estado de conversação e dados intermediários do onboarding ficará em memória no daemon do bot (`apps/bot`), encapsulado em um `SessionStore` protegido por `sync.RWMutex` com sessões indexadas pelo `telegram_id` / `chat_id`. — **Reversibilidade:** reversible — Mantém o bot desacoplado do banco e sem roundtrips HTTP intermediários na API.
- **D-02:** Estratégia de evicção de cache por TTL ativo implementada via `time.Ticker` em background no bot, executando periodicamente (a cada 5 a 10 minutos) e expurgando sessões inativas há mais de 30 minutos. — **Reversibilidade:** reversible
- **D-03:** Suporte ao comando `/cancelar` a qualquer momento durante o onboarding para resetar e expurgar a sessão em memória imediatamente. — **Reversibilidade:** reversible

### Entrada de Despesas Fixas Essenciais & Categorias
- **D-04:** Cadastro detalhado item a item das despesas fixas essenciais (ex.: `Aluguel 1500`, `Luz 250`). Cada item informado pelo usuário é estruturado como `FixedExpenseInput` e resulta na criação/vinculação automática de uma categoria/subcategoria de despesa fixa na API. — **Reversibilidade:** reversible
- **D-05:** A cada despesa fixa informada, o bot confirma o item na lista temporária e exibe um botão inline fixo `✅ Concluir Despesas Fixas` para que o usuário avance para a próxima etapa com 1 clique quando terminar. — **Reversibilidade:** reversible

### Recomendação de Reserva de Emergência & Aporte Mensal
- **D-06:** O bot calcula na hora o custo essencial mensal somando as despesas cadastradas e apresenta botões inline de 1 clique com os valores projetados: `🎯 6 Meses (CLT) - R$ X` vs `🎯 12 Meses (PJ) - R$ Y`. — **Reversibilidade:** reversible
- **D-07:** Imediatamente após a seleção dos meses de reserva, o bot pergunta o valor do aporte mensal planejado para essa reserva (`STATE_ONBOARDING_SAVINGS_TARGET`), alimentando o campo `target_savings` para dedução correta no cálculo do S2S diário. — **Reversibilidade:** reversible

### Etapa Opcional de Investimentos Iniciais
- **D-08:** Apresentação da etapa de investimentos via botões inline `➕ Adicionar Ativo` e `⏭️ Pular Etapa`. Caso o usuário opte por adicionar ativos, aceita entrada flexível (ex: `ALUP11 10 42.23`), atualiza a lista de investimentos e permite adicionar outros ou concluir. — **Reversibilidade:** reversible

### Comportamento para Usuários Já Cadastrados
- **D-09:** Ao receber `/start` de um usuário cujo setup já foi concluído anteriormente, o bot busca o contexto financeiro via `GET /internal/users/context-by-telegram` e exibe o Painel Diário com o S2S do dia, dias restantes no ciclo, badge de saúde (`SAUDÁVEL`, `RESTRITO`, etc.) e atalhos dos comandos principais (`/s2s`, `/gasto`, `/simular`, `/ajuda`). — **Reversibilidade:** reversible

### Validação de Dados & Formatação
- **D-10:** Parser de valores monetários tolerante a múltiplos formatos pt-BR (`3500.00`, `3500,00`, `3500`, `R$ 3.500,00`), convertendo com precisão para `decimal.Decimal` e evitando tipos float.
- **D-11:** Validação estrita do dia de início do ciclo entre 1 e 28, com mensagem amigável explicando que o teto é dia 28 para consistência de calendário com o mês de fevereiro.

### the agent's Discretion
- Formatação estética e emojis das mensagens de diálogo no Telegram.
- Nomes das constantes dos estados da FSM (`StateIdle`, `StateWaitingBalance`, `StateWaitingCycleDay`, etc.).
- Detalhes de implementação do parser regex de ativos e despesas fixas.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requisitos e Roadmap
- `docs/roadmap.md` §Milestone 2 — Especificação canônica do fluxo de Onboarding Conversacional no Telegram [v0.0.1]
- `.planning/ROADMAP.md` §Phase 1 — Metas, dependências, critérios de sucesso e planos da Fase 1
- `.planning/PROJECT.md` §Active Requirements M2-ONBOARD-01..07 — Requisitos ativos e decisões chave
- `.planning/REQUIREMENTS.md` §ONBD-01..07 — Critérios de aceitação rastreáveis do onboarding

### Contratos e Código Existente
- `apps/api/internal/botapi/onboarding.go` — Handler de `POST /internal/users/onboarding` e structs de request/response
- `apps/bot/internal/client/api_client.go` — Métodos `SaveOnboarding` e `GetUserContextByTelegram` já disponíveis
- `apps/bot/internal/bot/bot.go` — Entrada principal de mensagens e updates do Telegram Bot
- `apps/api/internal/dateinterval/date_interval.go` — Regras do ciclo financeiro dinâmico (dias 1 a 28)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `apps/bot/internal/client/api_client.go`: Possui `SaveOnboarding(ctx, req)` e `GetUserContextByTelegram(ctx, telegramID)` prontos para integração.
- `apps/api/internal/botapi/onboarding.go`: Já implementa o processamento transacional que cria o usuário, atualiza a conta bancária inicial, cria categorias de despesas fixas, cadastra investimentos e executa o motor preditivo retornando o primeiro S2S.
- `shopspring/decimal`: Biblioteca padrão do monorepo para manipulação exata de moedas e quantidades sem ponto flutuante.

### Established Patterns
- Separação estrita: `apps/bot` é cliente HTTP da `apps/api` via `X-Internal-Secret`. Não conecta diretamente ao banco PostgreSQL.
- Telegram Bot API wrapper: `go-telegram-bot-api/telegram-bot-api/v5`.
- Logging estruturado via `log/slog`.
- Mensagens de erro e retorno ao usuário estritamente em português (pt-BR).

### Integration Points
- `apps/bot/internal/bot/bot.go` (`handleMessage` e `handleUpdate`): Interceptação de mensagens de texto e de `CallbackQuery` (botões inline) para alimentar a máquina de estados.
- Novo subpacote `apps/bot/internal/bot/fsm` ou `apps/bot/internal/session`: Gerenciador de sessões e máquina de estados em memória.

</code_context>

<specifics>
## Specific Ideas

- Primeira mensagem após `/start` sem token de auth: Apresentar o conceito do Saldo Seguro Diário (S2S) e iniciar diretamente a pergunta do saldo em conta corrente.
- Botões inline para escolhas binárias ou listas curtas: `[ 🎯 6 Meses (CLT) ]` / `[ 🎯 12 Meses (PJ) ]`, `[ ✅ Concluir Despesas ]`, `[ ➕ Adicionar Ativo ]` / `[ ⏭️ Pular Investimentos ]`.
- Resumo final com visual limpo:
  ```text
  ✅ Configuração concluída com sucesso!

  💰 Saldo Seguro Diário (S2S): R$ 78,50/dia
  📅 Dias restantes no ciclo: 28 dias
  📊 Status do Ciclo: 🟢 SAUDÁVEL
  🛡️ Reserva de Emergência: Cobre 1,5 meses da sua meta de 6 meses
  ```

</specifics>

<deferred>
## Deferred Ideas

Nenhuma ideia diferida — todas as decisões e discussões mantiveram-se estritamente dentro do escopo do Milestone 2 / Fase 1.

</deferred>

---

*Phase: 1-Onboarding Conversacional no Telegram*
*Context gathered: 2026-10-03*
