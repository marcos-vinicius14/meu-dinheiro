# Phase 3: Gestão de Carteira de Ações no Telegram (`v0.1.0`) - Context

**Gathered:** 2026-10-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Entrega da gestão e acompanhamento consolidado da carteira de ações e ativos (incluindo ações da B3, FIIs, ETFs e suporte pragmático a títulos de Renda Fixa via tickers flexíveis) diretamente pelo bot do Telegram (`apps/bot`), cobrindo integralmente os requisitos `INVEST-01`, `INVEST-02` e `INVEST-03`:
- `/investimento <TICKER>, <QUANTIDADE> un a <PRECO>` (com aliases `/comprar`, `/aporte`): Adiciona novos lotes ao portfólio do usuário via `POST /internal/investments`, recalculando o Preço Médio ponderado com Banker's rounding HalfEven e quantidade total.
- `/carteira` (com aliases `/investimentos`, `/portfolio`): Exibe a carteira consolidada em cards visuais por ativo (2 linhas legíveis no mobile com emojis), ordenados por maior volume financeiro investido (R$), mostrando % de alocação de cada ativo, quantidade, PM e total investido, finalizando com o Total Investido e Patrimônio Líquido Global (saldo bancário + investimentos). Acompanhado por botões inline de ação rápida `[➕ Novo Aporte]` e `[🔄 Atualizar]`, além de card amigável para carteira vazia com botão `[➕ Adicionar Ativo]`. Exige novo endpoint `GET /internal/investments?telegram_id=...`.
- `/venda <TICKER> <QUANTIDADE> [a <PRECO>]` (com alias `/vender`): Realiza baixa de custódia com sintaxe flexível. Preço de venda opcional: se informado, calcula na hora o Lucro/Prejuízo realizado em R$ e % em relação ao PM, oferecendo botão inline de 1 clique para creditar o montante liquidado no saldo da conta corrente (`[💳 Creditar R$ X no Saldo Líquido]`). Validação de custódia com bloqueio de excesso e atalho `[Vender Todas as X]`. Card completo de liquidação. Exige novo endpoint `POST /internal/investments/sell`.
- Tratamento de posição zerada: Execução direta sem bloqueios com card de encerramento de ciclo do papel; botão inline efêmero `[↩️ Desfazer Venda]` para reversão imediata de erro de digitação; recompra futura inicia novo ciclo limpo (novo PM = preço da compra); ativos zerados são excluídos da listagem principal ativa.
- Teclado persistente 2x2 da Fase 2 (`[💰 S2S Hoje] [💸 Lançar Gasto] / [🔮 Simular] [📝 Check-in]`) preservado intacto.
- Envio de `/investimento` ou `/venda` sem parâmetros exibe card educativo com exemplos copiáveis (padrão `/gasto`).

</domain>

<decisions>
## Implementation Decisions

### Formatação e Apresentação da Carteira (`/carteira`, `/investimentos`)
- **D-01:** Formato em Cards por Ativo: A listagem da carteira utiliza blocos compactos de 2 linhas por papel com emojis temáticos, evitando quebra de tabela monoespaçada em telas de smartphones. Cada card destaca Ticker, % de alocação da carteira, quantidade de cotas, Preço Médio formatado e valor total investido. — **Reversibilidade:** reversible
- **D-02:** Ordenação por Maior Posição Financeira (R$): Os ativos são exibidos em ordem decrescente de valor total investido (`quantity * average_price`), permitindo ao investidor enxergar instantaneamente onde está concentrada a maior parte do seu patrimônio. — **Reversibilidade:** reversible
- **D-03:** Métricas Globais no Rodapé: O relatório da carteira encerra com o Total Geral Investido e o Patrimônio Líquido Global (somando o saldo líquido bancário em contas correntes ao total em custódia de investimentos), integrando o mundo de caixa diário (S2S) e o patrimônio acumulado. — **Reversibilidade:** reversible
- **D-04:** Tratamento de Carteira Vazia: Quando um usuário sem ativos cadastrados executar `/carteira`, o bot responde com mensagem encorajadora explicando como funciona o portfólio, exemplos de comandos e um botão inline `[➕ Adicionar Ativo]`. — **Reversibilidade:** reversible

### Sintaxe, Liquidação e Impacto de Venda (`/venda`)
- **D-05:** Sintaxe Flexível com Preço Opcional: O comando `/venda` aceita tanto `/venda TICKER QTD` (baixa simples de estoque) quanto `/venda TICKER QTD a PRECO` (ou `/venda TICKER QTD PRECO`). Quando o preço de venda for fornecido, o bot calcula o resultado da operação (`(Preço Venda - Preço Médio) * Qtd Vendida`) exibindo Lucro ou Prejuízo em R$ e rentabilidade percentual. — **Reversibilidade:** reversible
- **D-06:** Botão Inline Opcional de Crédito em Conta: Quando a venda apura caixa liquidado com preço de venda, o bot não altera o saldo bancário de forma oculta; em vez disso, exibe os botões inline `[💳 Creditar R$ X no Saldo Líquido]` (que registra receita interna via `RegisterIncome`) e `[🛡️ Manter Apenas na Carteira]`, garantindo controle total ao usuário sobre o impacto no seu S2S diário. — **Reversibilidade:** reversible
- **D-07:** Validação Defensiva de Custódia com Atalho de Venda Total: Se o usuário tentar vender uma quantidade superior à existente em carteira, a operação é recusada com mensagem clara informando a custódia atual e exibindo botão inline `[Vender Todas as X]` para liquidar a posição total sem precisar redigitar. — **Reversibilidade:** reversible
- **D-08:** Card Completo de Confirmação de Venda Parcial: A resposta de venda parcial detalha Ativo, Quantidade vendida, Preço praticado, Total apurado, Lucro/Prejuízo em R$ e %, e a Posição Remanescente (quantidade restante e novo valor de custódia). — **Reversibilidade:** reversible

### Tratamento de Posição Zerada e Ciclos
- **D-09:** Execução Direta com Card de Encerramento: Vendas que liquidam 100% da posição são executadas imediatamente sem fricção de confirmação prévia, exibindo um card especial de fechamento de ciclo destacando o lucro/prejuízo total apurado na operação. — **Reversibilidade:** reversible
- **D-10:** Botão Inline Efêmero de Reversão `[↩️ Desfazer Venda]`: Na confirmação de qualquer venda (parcial ou total), o bot inclui botão inline para desfazer imediatamente a operação caso o usuário tenha cometido erro de digitação de ticker ou quantidade. — **Reversibilidade:** reversible
- **D-11:** Recompra Inicia Novo Ciclo Limpo: No caso de recompra de ativo que havia sido previamente zerado, o sistema inicia um novo ciclo limpo onde o novo Preço Médio é estritamente o preço da nova compra (conforme legislação tributária da B3/Receita Federal), alertando o usuário sobre a abertura de nova posição. — **Reversibilidade:** reversible
- **D-12:** Ocultação de Posições Zeradas na Carteira: A listagem da `/carteira` exibe estritamente posições ativas (`quantity > 0`), mantendo a visão focada nos ativos sob custódia atual. — **Reversibilidade:** reversible

### Interface de Acesso, Atalhos e Aliases
- **D-13:** Preservação do Teclado Persistente 2x2: O teclado fixo inferior permanece dedicado às 4 operações diárias essenciais de caixa (`[💰 S2S Hoje] [💸 Lançar Gasto] / [🔮 Simular] [📝 Check-in]`); a carteira de investimentos opera via comandos de barra e botões inline contextuais. — **Reversibilidade:** reversible
- **D-14:** Botões Inline no Rodapé da Carteira: O relatório de `/carteira` inclui botões inline `[➕ Novo Aporte]` (orienta digitação do aporte) e `[🔄 Atualizar]` (recalcula e edita o card da carteira no mesmo lugar sem poluir o chat). — **Reversibilidade:** reversible
- **D-15:** Suporte Amplo a Aliases Naturais:
  - Consulta: `/carteira`, `/investimentos`, `/portfolio`
  - Aporte/Compra: `/investimento`, `/comprar`, `/aporte`
  - Venda/Baixa: `/venda`, `/vender`
  — **Reversibilidade:** reversible
- **D-16:** Feedback Educativo para Comandos sem Argumentos: Enviar `/investimento` ou `/venda` sem parâmetros não abre FSM conversacional longa; responde na hora com card explicativo e exemplos copiáveis no padrão do `/gasto` da Fase 2. — **Reversibilidade:** reversible

### Suporte Pragmático a Renda Fixa
- **D-17:** Renda Fixa via Tickers Flexíveis: Suporte pragmático a títulos públicos e bancários (Tesouro Selic, Tesouro IPCA, CDBs, LCIs) permitindo tickers alfanuméricos com hífen e ponto de até 12 caracteres (ex: `TD-SELIC`, `CDB-INTER`, `TD-IPCA29`), quantidade e preço unitário. O regex de ticker no bot (`apps/bot/internal/bot/fsm/parsers.go`) será alinhado com o regex da API (`^[A-Z0-9.\-_]{1,12}$`). — **Reversibilidade:** reversible

### the agent's Discretion
- Emojis exatos e diagramação visual dos cards no Telegram.
- Estruturação do DTO nos novos endpoints internos da API (`GET /internal/investments` e `POST /internal/investments/sell`).
- Implementação interna da reversão de venda no handler de callback (`[↩️ Desfazer Venda]`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requisitos e Roadmap
- `docs/roadmap.md` §Milestone 4 — Carteira de Ações e Investimentos no Telegram [v0.1.0]
- `.planning/ROADMAP.md` §Phase 3 — Gestão de Carteira de Ações no Telegram (`v0.1.0`)
- `.planning/PROJECT.md` §v2 Requirements / Key Decisions
- `.planning/REQUIREMENTS.md` §INVEST-01..03 — Requisitos rastreáveis da carteira de ações e investimentos

### Contratos e Código Existente
- `apps/api/internal/investment/investment.go` — Regras puras de Preço Médio ponderado (`CalculateNewAveragePrice`) e baixa de venda (`CalculateSale`)
- `apps/api/internal/investment/repository.go` — Consultas SQL e transações com locking `FOR UPDATE` para `AddLot` e `Sell`
- `apps/api/internal/investment/service.go` — Métodos `AddLot`, `Sell`, `List` prontos para exposição no `botapi`
- `apps/api/internal/botapi/investment.go` — Handler existente de `POST /internal/investments` e novas rotas a adicionar (`GET /internal/investments`, `POST /internal/investments/sell`)
- `apps/api/internal/botapi/handler.go` — Registro das novas rotas internas autenticadas por `X-Internal-Secret`
- `apps/bot/internal/client/api_client.go` — Métodos HTTP do cliente do bot (`AddInvestment` existente; adicionar `ListInvestments` e `SellInvestment`)
- `apps/bot/internal/bot/bot.go` — Dispatcher de comandos `/carteira`, `/investimento`, `/venda` e callbacks inline
- `apps/bot/internal/bot/fsm/parsers.go` — Parsers numéricos e expansão do `tickerRegex` para suportar até 12 caracteres com hífen/ponto

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `apps/api/internal/investment/service.go`: Já implementa `AddLot`, `Sell` e `List` completos, testados com precisão de arredondamento bancário (`money.Money`).
- `apps/api/internal/investment/repository.go`: Possui query transacional com `SELECT ... FOR UPDATE`, cálculo de `CalculateSale` e deleção automática quando a quantidade zera.
- `apps/api/internal/botapi/investment.go`: Já possui `POST /internal/investments` funcional consumindo `AddLot`.
- `apps/bot/internal/client/api_client.go`: Já possui método `AddInvestment`.
- `apps/bot/internal/bot/fsm/parsers.go`: Já possui `ParseInvestment` com suporte a formatos com `un`, `cotas`, `a`, `de`.

### Established Patterns
- Separação estrita: `apps/bot` consome `apps/api` via HTTP interno com cabeçalho `X-Internal-Secret`.
- Toda manipulação monetária usa `money.Money` com Banker's rounding HalfEven (`decimal.Decimal`).
- Respostas da API e do bot 100% em português (pt-BR).
- Testes reais de integração contra PostgreSQL 18 via Testcontainers (zero mocks).
- Respostas rápidas e educativas com exemplos práticos quando comandos são enviados sem argumentos.

### Integration Points
- `apps/api/internal/botapi/investment.go`: Adicionar handlers para `GET /internal/investments?telegram_id=...` e `POST /internal/investments/sell`.
- `apps/api/internal/botapi/handler.go`: Registrar rotas no grupo interno do Chi.
- `apps/bot/internal/client/api_client.go`: Adicionar métodos `ListInvestments` e `SellInvestment`.
- `apps/bot/internal/bot/bot.go`: Registrar comandos `/carteira`, `/investimentos`, `/portfolio`, `/investimento`, `/comprar`, `/aporte`, `/venda`, `/vender` e callbacks inline `invest:aporte`, `invest:refresh`, `invest:credit_income`, `invest:undo_sell`.
- `apps/bot/internal/bot/fsm/parsers.go`: Ajustar `tickerRegex` para `^[A-Z0-9.\-_]{1,12}$` e criar `ParseSaleCommand`.

</code_context>

<specifics>
## Specific Ideas

### Exemplo do Card da Carteira (`/carteira`)
```text
📈 Sua Carteira de Investimentos

🔹 ALUP11 (35,2% da carteira)
• 100 cotas • PM: R$ 42,23 • Total: R$ 4.223,00

🔹 PETR4 (28,5% da carteira)
• 80 cotas • PM: R$ 38,50 • Total: R$ 3.080,00

🔹 TD-SELIC (36,3% da carteira)
• 1 cota • PM: R$ 14.500,00 • Total: R$ 14.500,00
───────────────────────────
💼 Total Investido: R$ 21.803,00
🏦 Saldo em Conta: R$ 3.197,00
🌐 Patrimônio Líquido Total: R$ 25.000,00

[ ➕ Novo Aporte ] [ 🔄 Atualizar ]
```

### Exemplo de Confirmação de Venda Parcial com Preço (`/venda PETR4 30 a 41.50`)
```text
✅ Venda registrada com sucesso!

📉 Ativo: PETR4
• Vendido: 30 cotas a R$ 41,50
• Total Apurado: R$ 1.245,00
• Preço Médio: R$ 38,50
• Lucro Realizado: +R$ 90,00 (+7,79%) 🟢

📊 Custódia Remanescente: 50 cotas (R$ 1.925,00)

[ 💳 Creditar R$ 1.245,00 no Saldo Líquido ] [ 🛡️ Manter Apenas na Carteira ]
[ ↩️ Desfazer Venda ]
```

### Exemplo de Encerramento de Posição Zerada (`/venda PETR4 80 a 41.50`)
```text
🏁 Posição Encerrada!

Você vendeu 100% das suas cotas de PETR4.
• Total Apurado: R$ 3.320,00
• Lucro total realizado: +R$ 240,00 (+7,79%) 🟢
• PETR4 foi removido da sua custódia ativa.

[ 💳 Creditar no Saldo Líquido ] [ ↩️ Desfazer Venda ]
```

</specifics>

<deferred>
## Deferred Ideas

- **Módulo Avançado de Renda Fixa:** Modelagem completa de ativos de renda fixa com indexadores dinâmicos (% CDI, IPCA + taxa pré), datas de vencimento, liquidez, marcação a mercado e apuração de IR regressivo e come-cotas. Mantido no backlog de roadmap para fase futura.

</deferred>

---

*Phase: 3-Gestão de Carteira de Ações no Telegram (`v0.1.0`)*
*Context gathered: 2026-10-04*
