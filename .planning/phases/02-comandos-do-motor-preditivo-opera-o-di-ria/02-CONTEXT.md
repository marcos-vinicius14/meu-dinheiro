# Phase 2: Comandos do Motor Preditivo & Operação Diária - Context

**Gathered:** 2026-10-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Disponibilização no Telegram Bot (`apps/bot`) dos comandos centrais do motor preditivo para a rotina diária de controle financeiro do usuário, cobrindo integralmente os requisitos `PRED-01` a `PRED-04`:
- `/s2s`: Consulta instantânea do Saldo Seguro Diário (S2S), dias restantes no ciclo mensal, limite flexível restante e badge de saúde financeira (`🟢 SAUDÁVEL`, `🟡 RESTRITO`, `🔴 RISCO DE DÉFICIT`).
- `/gasto <valor> <descrição>`: Registro rápido de despesas flexíveis avulsas com recálculo imediato do S2S e feedback comparativo (antes vs depois) e seleção de categoria via botões inline.
- `/simular <valor> [parcelas]`: Simulador what-if multi-ciclo projetando até 12 ciclos futuros, destacando o ciclo mais crítico (menor S2S), emitindo alertas preventivos de risco de déficit com recomendações de ajuste e botão inline `[✅ Confirmar e Lançar]` para conversão em transação real.
- `/checkin`: Conciliação diária guiada de gastos e fechamento do dia via FSM interativa, permitindo lançamento de despesas esquecidas, conferência e ajuste opcional de saldo bancário líquido, gerando o relatório de economia do dia e gravando o snapshot diário em `tb_check_in_snapshots` de forma idempotente (`UPSERT`).
- Teclado persistente (`ReplyKeyboardMarkup`) com grade compacta 2x2 (`[💰 S2S Hoje] [💸 Lançar Gasto] / [🔮 Simular] [📝 Check-in]`) integrado após o onboarding e no `/start` de usuários já cadastrados.

</domain>

<decisions>
## Implementation Decisions

### Feedback e Apresentação do S2S e Gastos (`/s2s`, `/gasto`)
- **D-01:** O comando `/s2s` responde com um Card Completo de Ciclo contendo: S2S diário formatado (`R$ XX,XX/dia`), badge de saúde financeira com emoji (`🟢 SAUDÁVEL`, `🟡 RESTRITO`, `🔴 RISCO DE DÉFICIT`), contagem de dias restantes no ciclo, saldo flexível disponível no ciclo e total já gasto até o momento. — **Reversibilidade:** reversible
- **D-02:** O comando `/gasto <valor> <descrição>` fornece feedback comparativo imediato destacando o gasto confirmado, o S2S anterior vs novo S2S, a variação diária (`-R$ X,XX/dia`) e os dias restantes no ciclo. — **Reversibilidade:** reversible
- **D-03:** Se o usuário enviar `/gasto` sem argumentos, o bot não inicia um diálogo longo; responde imediatamente com a explicação da sintaxe e exemplos práticos (ex: `/gasto 34.90 Almoço`, `/gasto 120 Mercado`) para reenvio direto. — **Reversibilidade:** reversible
- **D-04:** No `/gasto`, após identificar a transação, o bot apresenta botões inline com as categorias cadastradas do usuário para confirmação rápida em 1 clique (com fallback automático pela API caso a categoria seja explicitamente inferida ou já informada). — **Reversibilidade:** reversible

### Visualização da Simulação What-If (`/simular`)
- **D-05:** O comando `/simular` exibe um Resumo Executivo focado no impacto imediato no ciclo atual, valor de cada parcela e destaque do ciclo mais crítico (menor S2S dos 12 meses futuros) para evitar poluição visual de uma tabela inteira de 12 meses no chat do Telegram. — **Reversibilidade:** reversible
- **D-06:** Sintaxe flexível: aceita `/simular <valor>` assumindo compra à vista (1 parcela) ou `/simular <valor> <parcelas>` (ex: `/simular 2400 12` para 12x de R$ 200,00). — **Reversibilidade:** reversible
- **D-07:** Resposta da simulação inclui botão inline `[✅ Confirmar e Lançar]` que permite converter a simulação em compra real (como gasto flexível ou bundle parcelado) diretamente pelo chat sem ter que redigitar. — **Reversibilidade:** reversible
- **D-08:** Quando a simulação detectar risco de déficit futuro em algum ciclo, dispara alerta explícito com diagnóstico detalhado indicando o ciclo exato e recomendando ações corretivas (ex: "Ciclo 3 ficará em Déficit: considere 15x ou adiar"). — **Reversibilidade:** reversible

### Mecânica do Check-in Diário (`/checkin`)
- **D-09:** O `/checkin` implementa um diálogo guiado em mini-FSM: inicia perguntando se o usuário teve outras despesas no dia não registradas no bot, com botões inline `[Sim, lançar]` e `[Não, tudo certo]`. — **Reversibilidade:** reversible
- **D-10:** Verificação de sanidade do saldo bancário: o bot exibe o saldo líquido calculado das contas correntes e oferece botões inline `[✅ Sim, confere]` e `[✏️ Ajustar saldo]`. Se houver divergência, permite atualizar o saldo das contas bancárias. — **Reversibilidade:** costly — Envolve atualização de saldo bancário na API e recálculo de liquidez.
- **D-11:** Ao concluir o check-in, o bot envia o Relatório de Conquista Diária mostrando o gasto realizado no dia vs a meta diária (S2S), economia gerada no dia que aumenta o S2S de amanhã e o status geral do ciclo. — **Reversibilidade:** reversible
- **D-12:** Gravação do snapshot de check-in é idempotente via `UPSERT` em `tb_check_in_snapshots`. Executar o `/checkin` mais de uma vez na mesma data reavalia e atualiza o snapshot daquele dia sem duplicatas. — **Reversibilidade:** reversible

### Interface de Acesso aos Comandos (Teclado Persistente)
- **D-13:** Teclado persistente (`ReplyKeyboardMarkup`) em grade compacta 2x2 com os 4 comandos centrais de operação:
  `[ 💰 S2S Hoje ] [ 💸 Lançar Gasto ]`
  `[ 🔮 Simular ] [ 📝 Check-in ]`
  com `ResizeKeyboard: true`. — **Reversibilidade:** reversible
- **D-14:** O teclado persistente é ativado automaticamente na mensagem final do onboarding e sempre que um usuário já cadastrado enviar `/start`. — **Reversibilidade:** reversible
- **D-15:** Ao tocar em `[ 💰 S2S Hoje ]` ou `[ 📝 Check-in ]`, a execução é direta e instantânea (equivalente a digitar `/s2s` ou `/checkin`). Ao tocar em `[ 💸 Lançar Gasto ]` ou `[ 🔮 Simular ]`, o bot envia mensagem orientadora com exemplos de formato para digitação rápida. — **Reversibilidade:** reversible

### the agent's Discretion
- Emojis, espaçamentos e formatações exatas das mensagens em Markdown no Telegram.
- Estruturação dos novos endpoints internos na API Go (`POST /internal/transactions/simulations` e `POST /internal/transactions/checkin`) para consumo pelo `client.APIClient`.
- Nomes dos novos estados da FSM para o fluxo guiado do check-in (`StateCheckinWaitingExpenses`, `StateCheckinWaitingBalanceConfirm`, etc.).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requisitos e Roadmap
- `docs/roadmap.md` §Milestone 3 — Operação Diária & Motor Preditivo no Telegram [v0.0.1]
- `.planning/ROADMAP.md` §Phase 2 — Metas, dependências, critérios de sucesso e planos da Fase 2
- `.planning/PROJECT.md` §Active Requirements M3-PRED-01..04 — Requisitos ativos e decisões chave
- `.planning/REQUIREMENTS.md` §PRED-01..04 — Critérios de aceitação rastreáveis da operação diária

### Contratos e Código Existente
- `apps/api/internal/botapi/handler.go` — Handler base de rotas internas consumidas pelo bot
- `apps/api/internal/botapi/quick_expense.go` — Implementação existente de `POST /internal/transactions/quick-expense`
- `apps/api/internal/botapi/context.go` — Implementação de `GET /internal/users/context-by-telegram`
- `apps/api/internal/transaction/service.go` — Métodos `SimulatePurchase` e `DailyCheckIn` prontos para exposição interna
- `apps/api/internal/transaction/engine/predictive_engine.go` — Motor de cálculo do S2S diário e status de saúde
- `apps/api/internal/transaction/engine/what_if_simulator.go` — Simulador what-if multi-ciclo
- `apps/bot/internal/client/api_client.go` — Métodos HTTP do bot (`QuickExpense`, `GetUserContextByTelegram`) e novos métodos a adicionar
- `apps/bot/internal/bot/bot.go` — Despacho de mensagens e roteamento de comandos no bot
- `apps/bot/internal/bot/fsm/` — Máquina de estados, store de sessões em memória e handlers

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `apps/api/internal/transaction/service.go`: Possui `SimulatePurchase` e `DailyCheckIn` implementados e testados, faltando apenas expô-los em rotas internas do `botapi` (`POST /internal/transactions/simulations` e `POST /internal/transactions/checkin`).
- `apps/bot/internal/client/api_client.go`: Já possui `QuickExpense` e `GetUserContextByTelegram`. Bastará adicionar `SimulatePurchase` e `DailyCheckIn`.
- `apps/bot/internal/bot/fsm`: Estrutura de máquina de estados em memória com `SessionStore` e `StepHandler` pronta para reutilização no fluxo guiado do `/checkin`.
- `apps/bot/internal/bot/fsm/parsers.go`: Parsers numéricos e monetários tolerantes com `decimal.Decimal` prontos para extrair valores de `/gasto` e `/simular`.

### Established Patterns
- Separação estrita: `apps/bot` é cliente HTTP da `apps/api` via cabeçalho `X-Internal-Secret`.
- Erros da API traduzidos com mensagens claras em português (pt-BR).
- `money.Money` com Banker's rounding HalfEven (`decimal.Decimal`) para todas as manipulações monetárias.
- Logging estruturado via `log/slog`.

### Integration Points
- `apps/api/internal/botapi/handler.go`: Registro dos endpoints `POST /internal/transactions/simulations` e `POST /internal/transactions/checkin`.
- `apps/bot/internal/client/api_client.go`: Métodos `SimulatePurchase` e `DailyCheckIn`.
- `apps/bot/internal/bot/bot.go`: Interceptação dos comandos `/s2s`, `/gasto`, `/simular`, `/checkin` e botões de atalho do teclado persistente.
- `apps/bot/internal/bot/fsm/`: Novos estados da FSM para conduzir a conciliação guiada do check-in.

</code_context>

<specifics>
## Specific Ideas

- **Exemplo de Card `/s2s`:**
  ```text
  💰 Saldo Seguro Diário (S2S): R$ 78,50/dia
  📊 Status do Ciclo: 🟢 SAUDÁVEL
  📅 Dias restantes no ciclo: 22 dias
  💳 Limite Flexível restante: R$ 1.727,00
  📉 Total gasto no ciclo: R$ 850,00
  ```

- **Exemplo de Feedback `/gasto 35 Almoço`:**
  ```text
  ✅ Gasto de R$ 35,00 registrado em Alimentação!

  📉 S2S Anterior: R$ 85,00/dia
  💰 Novo S2S: R$ 78,50/dia (-R$ 6,50/dia)
  📅 Dias restantes: 22 dias
  📊 Status: 🟢 SAUDÁVEL
  ```

- **Exemplo de Resumo `/simular 2400 12`:**
  ```text
  🔮 Simulação de Compra: R$ 2.400,00 (12x de R$ 200,00)

  📅 Impacto no Ciclo Atual: S2S cai de R$ 78,50 para R$ 71,80/dia
  ⚠️ Ciclo Mais Crítico: Ciclo 3 (S2S de R$ 32,10/dia - 🟡 RESTRITO)
  💡 Recomendação: Compra viável, mas o Ciclo 3 exigirá disciplina com gastos flexíveis.

  [ ✅ Confirmar e Lançar ] [ ❌ Cancelar ]
  ```

- **Exemplo de Encerramento `/checkin`:**
  ```text
  🏁 Check-in Diário Concluído!

  📊 Resumo de Hoje:
  • Gastos realizados: R$ 42,00
  • Cota do dia (S2S): R$ 78,50
  🎉 Você economizou R$ 36,50 hoje!

  💰 S2S Projetado para Amanhã: R$ 80,15/dia (+R$ 1,65)
  🟢 Ciclo segue SAUDÁVEL. Até amanhã!
  ```

</specifics>

<deferred>
## Deferred Ideas

Nenhuma ideia diferida — todas as decisões e discussões mantiveram-se estritamente dentro do escopo do Milestone 3 / Fase 2.

</deferred>

---

*Phase: 2-Comandos do Motor Preditivo & Operação Diária*
*Context gathered: 2026-10-03*
