# Summary 03-02: Comandos de Carteira, Aporte e Venda com Callbacks Inline e Suporte a Renda Fixa no Bot Telegram

## Resumo da Execução

A Wave 2 da Fase 3 foi concluída com sucesso em conformidade com as diretrizes de código idiomático Go, arquitetura modular sem God Files (handler dedicado `invest_handler.go`), preservação do teclado persistente 2x2 e cobertura completa de testes unitários com `-race`.

### 1. Camada de Parsing (`fsm/parsers.go`)
- **Expansão de Sintaxe de Tickers (D-17)**:
  - `tickerRegex` expandido de `^[A-Z0-9]{4,6}$` para `^[A-Z0-9.\-_]{1,12}$`, habilitando ativos da B3 (ex: `BOVA11`, `PETR4`, `ALUP11`) e títulos de Renda Fixa (ex: `TD-SELIC`, `CDB-INTER`, `TD-IPCA29`).
- **Parser de Venda Flexível (`ParseSaleCommand`) (D-05, T-03-08)**:
  - Suporta comandos com preço (`/venda PETR4 30 a 41.50`, `/venda PETR4 30 41.50`, `/venda PETR4, 30 un a 41,50`) e sem preço (`/venda PETR4 30`).
  - Validação estrita de quantidade (`qty > 0`) e preço não-negativo (`price >= 0`).

### 2. Handler de Apresentação e Interação (`invest_handler.go`)
- **Carteira Consolidada (`handlePortfolioCommand`)**:
  - Cards de 2 linhas por papel com emojis temáticos (D-01).
  - Ordenação por maior volume financeiro total investido (R$) decrescente (D-02).
  - Cálculo de % de alocação de cada ativo sobre o total investido (D-01).
  - Rodapé consolidado com Total Investido, Saldo em Conta e Patrimônio Líquido Total (D-03).
  - Empty state amigável com botão inline `[➕ Adicionar Ativo]` (D-04).
  - Botões inline `[➕ Novo Aporte]` e `[🔄 Atualizar]` com in-place edit (`tgbotapi.NewEditMessageTextAndMarkup`) no refresh para evitar poluição do chat (D-14).
- **Aporte em Carteira (`handleInvestCommand`) (INVEST-01)**:
  - Suporte a `/investimento`, `/comprar` e `/aporte` (D-15).
  - Resposta imediata com card educativo copiável quando invocado sem argumentos (D-16).
  - Recálculo de Preço Médio Ponderado e apresentação de posição atualizada.
- **Venda e Baixa de Custódia (`handleSellCommand`) (INVEST-03)**:
  - Suporte a `/venda` e `/vender` (D-15).
  - Resposta com card educativo copiável quando invocado sem argumentos (D-16).
  - Apuração de Lucro/Prejuízo realizado em R$ e % com badge visual (`🟢 Lucro` / `🔴 Prejuízo`) quando informado preço de venda (D-05, D-08).
  - Botão inline opcional `[💳 Creditar R$ X no Saldo Líquido]` via `RegisterIncome` e `[🛡️ Manter Apenas na Carteira]` (D-06).
  - Atalho inteligente de excesso de custódia `[Vender Todas as X]` consultando a posição atual (D-07).
  - Encerramento de posição zerada com card especial `🏁 Posição Encerrada!` (D-09).
  - Botão efêmero de reversão `[↩️ Desfazer Venda]` restaurando as cotas ao PM original (D-10).
- **Callbacks Inline (`handleInvestCallback`)**:
  - Tratamento seguro de `invest:aporte`, `invest:refresh`, `invest:credit`, `invest:keep`, `invest:undo`, `invest:sell_all`, `invest:cancel`.

### 3. Integração no Bot (`bot.go` e `fsm.go`)
- Adicionado roteamento sem colisões no `b.handleMessage` e `b.handleCallbackQuery`.
- Adicionada delegação explícita do prefixo `invest:` na FSM para permitir callbacks inline de investimentos sem conflito com o fluxo de onboarding.
- Adicionadas explicações de investimentos em `/ajuda` e `/start`.
- Teclado persistente 2x2 inferior preservado integralmente para as 4 operações diárias de caixa (D-13).

## Verificação Automatizada
- `cd apps/bot && go test -v ./internal/bot/fsm/... -run 'TestParse(Investment|SaleCommand)'`: 100% PASS.
- `cd apps/bot && go test -v -race ./internal/bot/... -run 'Test(Invest|Portfolio|Sell)'`: 100% PASS sem data races.
- `make test-bot`: 100% PASS.
- `make test`: 100% PASS em todo o monorepo.
- `make build-all`: 100% PASS (binários de api, bot e web construídos com sucesso).
