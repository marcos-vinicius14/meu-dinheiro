# 02-03-SUMMARY: Simulação What-If 12 Ciclos, Alerta Preventivo de Déficit e Conciliação Guiada de Check-in Diário

## Visão Geral

Conclusão da terceira e última onda da Fase 2 (Comandos do Motor Preditivo & Operação Diária), disponibilizando os dois recursos de inteligência financeira preditiva mais avançados no Telegram Bot (`apps/bot`):
- O comando `/simular` para projeções what-if multi-ciclo de até 12 meses com Resumo Executivo (impacto imediato + ciclo gargalo), alerta preventivo de déficit com recomendações de ajuste e botão inline `[ ✅ Confirmar e Lançar ]` para conversão com 1 clique em compra real.
- O comando `/checkin` operado através de um diálogo conversacional guiado na FSM com levantamento de despesas não registradas, conferência e ajuste de saldo bancário com 1 clique, gravação idempotente do snapshot diário (`UPSERT`) e entrega do Relatório de Conquista Diária com a economia acumulada e S2S recalculado para o dia seguinte.

---

## Entregas Realizadas

### 1. Parser de Simulação (`fsm/parsers.go` e `parsers_test.go`)
- Função `ParseSimulationCommand`:
  - Aceita compras à vista (`/simular 1500` -> 1 parcela implícita) e compras parceladas (`/simular 2400 12`).
  - Suporta formatos monetários pt-BR (`R$ 3.500,00 10`, `450.50 3`, `1200`).
  - Validação estrita de limites: parcelas entre 1 e 48, valores estritamente positivos e mensagens de validação em português.
  - Testes unitários com 100% de cobertura de casos nominais e de borda em `parsers_test.go`.

### 2. Comando `/simular` e Alerta Preventivo de Déficit (`simular_handler.go`, `bot.go`, `bot_simular_test.go`)
- Interceptação de `/simular` e do botão `[ 🔮 Simular ]` (`ButtonSimulate`):
  - Sem parâmetros ou ao tocar no botão: devolve guia explicativo com sintaxe e 3 exemplos práticos sem travar o diálogo (D-15).
  - Com parâmetros: aciona o endpoint interno de simulação na API (`POST /internal/transactions/simulations`).
- **Resumo Executivo (D-05, D-06)**:
  - Formatação concisa evitando sobrecarga de mensagens no chat.
  - Exibe: valor da compra e valor de cada parcela, impacto no ciclo atual (queda do S2S de antes vs depois com `-R$ X,XX/dia`) e o ciclo mais crítico entre os 12 meses (número do ciclo, S2S projetado e badge de saúde).
- **Alerta Preventivo de Déficit (D-08)**:
  - Se a compra projetada levar qualquer ciclo futuro ao vermelho (`DeficitRiskAlert = true` ou status `DEFICIT`/`DEFICIT_RISK`):
  - O bot destaca o alerta em vermelho (`🚨 ALERTA DE RISCO DE DÉFICIT:`) detalhando qual ciclo ficará negativo e entregando uma recomendação inteligente para reequilibrar o orçamento (ex: sugerindo número maior de parcelas).
- **Conversão em Compra Real (D-07)**:
  - Teclado inline com botões `[ ✅ Confirmar e Lançar ]` (callback `sim_confirm:<amount>:<installments>`) e `[ ❌ Cancelar ]` (`sim_cancel`).
  - Ao confirmar, o bot registra a despesa via `QuickExpense` na API e notifica a efetivação informando que o S2S foi atualizado.

### 3. Diálogo Guiado de Check-in Diário (`checkin_handler.go`, `fsm.go`, `types.go`, `checkin_handler_test.go`)
- Novos estados da FSM dedicados ao fluxo diário:
  - `StateCheckinWaitingExpenses`, `StateCheckinEnteringExpense`, `StateCheckinWaitingBalanceConfirm`, `StateCheckinEnteringBalance`.
- Ações e botões inline estruturados:
  - `checkin_add_expense`, `checkin_no_expenses`, `checkin_finish_expenses`, `checkin_balance_ok`, `checkin_balance_adjust`.
- Etapas do fluxo guiado:
  - **Início**: acionado por `/checkin` ou botão `[ 📝 Check-in ]`. Carrega saldo líquido atual do usuário e consulta se houve despesas não lançadas no dia (D-09).
  - **Lançamento Opcional de Despesas**: botão `[ ➕ Sim, lançar ]` permite entrada rápida e acumulada no formato `<valor> <descrição>`, com botão `[ ✅ Concluir Despesas ]` para avançar.
  - **Conferência de Saldo Bancário (D-10)**: exibe o saldo líquido calculado e oferece confirmação em 1 clique `[ ✅ Sim, confere ]` ou ajuste rápido `[ ✏️ Ajustar saldo ]`.
  - **Ajuste de Saldo**: botão `[ ✏️ Ajustar saldo ]` aceita o saldo real atual das contas e atualiza o saldo bancário via `AdjustAccountBalance`.
  - **Fechamento e Relatório de Conquista Diária (D-11, D-12)**: chama `DailyCheckIn` na API para gravar o snapshot diário de forma idempotente (`UPSERT`) e entrega o relatório diário:
    - Gastos do dia vs Cota do dia (S2S).
    - Economia gerada hoje (`🎉 Você economizou R$ X,XX hoje!`).
    - S2S projetado para amanhã com bônus diário (`+R$ X,XX/dia`).
    - Status de saúde do ciclo e mensagem motivacional.

---

## Verificação e Qualidade

- `cd apps/bot && go test -v -race ./internal/bot/... -run TestSimular` -> **PASS** (100% verde com detecção de races).
- `cd apps/bot && go test -v -race ./internal/bot/fsm/... -run "TestParseSimulationCommand|TestCheckin"` -> **PASS** (100% verde com detecção de races).
- `make test-bot` -> **PASS** (100% verde).
- `make test-api` -> **PASS** (todos os testes de integração com Postgres 18 via Testcontainers passando 100%).
- `make test` -> **PASS** (suíte completa do monorepo verde).
- `make build-all` -> **PASS** (binários de api, bot e build web compilam perfeitamente).
- Análise estática e formatação: `go vet` e `gofmt -s -w .` sem nenhum aviso.
