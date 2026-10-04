# Phase 3: Gestão de Carteira de Ações no Telegram (v0.1.0) - Research

**Phase Number:** 03  
**Phase Name:** Gestão de Carteira de Ações no Telegram (`v0.1.0`)  
**Status:** Completed  
**Requirements Covered:** `INVEST-01`, `INVEST-02`, `INVEST-03`  
**Relevant Context & Decisions:** Decisions `D-01` through `D-17` from `03-CONTEXT.md`

---

## 1. Executive Summary & Goals

Phase 3 delivers full investment and equity portfolio tracking through the Telegram Bot (`apps/bot`) backed by atomic transactional endpoints in the Go REST API (`apps/api`). 

### Core User Capabilities Delivered
1. **Asset Contribution & PM Recalculation (`INVEST-01`, `D-15`, `D-16`, `D-17`):**
   - Command `/investimento <TICKER> <QUANTIDADE> [a] <PREÇO>` (aliases: `/comprar`, `/aporte`).
   - Recalculates weighted average price (Preço Médio Ponderado - PM) using Banker's Rounding (HalfEven) to 2 decimal places and cumulative custody quantity.
   - Broadened ticker syntax (`^[A-Z0-9.\-_]{1,12}$`) supporting B3 stocks, FIIs, ETFs, and fixed income assets (`TD-SELIC`, `CDB-INTER`, `TD-IPCA29`).
   - Educational card returned when invoked without arguments (matching the pattern established for `/gasto`).
2. **Consolidated Portfolio Visualization (`INVEST-02`, `D-01`..`D-04`, `D-12`, `D-14`):**
   - Command `/carteira` (aliases: `/investimentos`, `/portfolio`).
   - Compact visual cards (2 readable lines per asset on mobile with emojis) sorted strictly by total financial volume invested descending (`quantity * average_price` DESC).
   - Metrics: allocation % of each asset within total invested, shares/units count, PM, and total asset cost.
   - Portfolio footer integrating total invested, current liquid bank balance (`totalLiquid`), and global net worth (`totalNetWorth = totalLiquid + totalInvested`).
   - Inline action buttons `[➕ Novo Aporte]` and `[🔄 Atualizar]` (editing message in-place without chat pollution).
   - Encouraging empty-state card with copyable examples and inline button `[➕ Adicionar Ativo]` when portfolio has no active positions.
   - Zero-quantity positions are filtered out from the active portfolio view.
3. **Custody Reduction, Sale Liquidation & Reversal (`INVEST-03`, `D-05`..`D-11`, `D-15`, `D-16`):**
   - Command `/venda <TICKER> <QUANTIDADE> [a <PREÇO>]` (alias: `/vender`).
   - Preço de venda opcional:
     - When sale price is provided: calculates realized Profit/Loss (Lucro/Prejuízo) in R$ and % relative to PM. Offers inline buttons `[💳 Creditar R$ X no Saldo Líquido]` (calls `RegisterIncome` to credit proceeds directly to bank balance and adjust S2S) and `[🛡️ Manter Apenas na Carteira]`.
     - When sale price is omitted: executes simple custody reduction, keeps PM intact, confirms remaining position without financial P&L.
   - Custody validation with shortcut: If user requests selling more shares than currently owned, rejects with clear explanation showing current custody and offers 1-click inline button `[Vender Todas as X]` to liquidate 100% of the position.
   - Zero-position closure: 100% sales execute immediately without friction, show a position closed card, delete the row in `tb_investments`, and future repurchases cleanly restart a new cycle where PM equals the new purchase price.
   - Typo safety: Every sale confirmation card provides an ephemeral inline button `[↩️ Desfazer Venda]` to immediately repurchase the sold lot at PM.

---

## 2. API Architecture & Endpoint Specification (`apps/api`)

### 2.1 Existing Domain & Persistence Audit (`internal/investment`)
The core domain model and repository in `apps/api/internal/investment` already implement high-quality financial primitives:
- **`CalculateNewAveragePrice`**: Uses `decimal.Decimal` and `money.Money`. New PM formula:
  $$\text{Novo PM} = \text{RoundBank}\left(\frac{(\text{Qtd Atual} \times \text{PM Atual}) + (\text{Qtd Nova} \times \text{Preço Novo})}{\text{Qtd Total}}, 2\right)$$
- **`CalculateSale`**: Maintains current average price intact. Decrements quantity. If remaining quantity is zero, sets `isClosed = true`.
- **`repository.AddLot`**: Uses transaction `database.WithTx` with `SELECT ... FOR UPDATE` row locking on `(user_id, ticker)` to guarantee absolute consistency under concurrent requests.
- **`repository.Sell`**: Uses `SELECT ... FOR UPDATE`. When `isClosed == true`, it deletes the row from `tb_investments`, which ensures that:
  1. Closed positions are never returned in subsequent `ListByUserID` queries (`D-12`).
  2. Future repurchases hit `errors.Is(err, pgx.ErrNoRows)` in `AddLot`, initializing a clean new cycle where the new PM is strictly the new purchase price (`D-11`).
- **Sentinels**:
  - `ErrInvestmentNotFound = errors.New("investimento não encontrado")` (maps to HTTP 404).
  - `ErrInvalidTicker = errors.New("código do ativo inválido. Deve ter entre 1 e 12 caracteres alfanuméricos")` (maps to HTTP 400).
  - `ErrInvalidQuantity = errors.New("quantidade deve ser maior que zero")` (maps to HTTP 400).
  - `ErrInvalidPrice = errors.New("preço deve ser maior ou igual a zero")` (maps to HTTP 400).
  - `ErrInsufficientQuantity = errors.New("quantidade insuficiente para venda")` (maps to HTTP 400).

### 2.2 New Internal Endpoints in `apps/api/internal/botapi/investment.go`

Two new endpoints must be implemented in `apps/api/internal/botapi/investment.go`:

#### Endpoint 1: `GET /internal/investments?telegram_id={id}`
- **Purpose:** Retrieve the user's active investment portfolio along with liquid cash and net worth for Telegram card rendering.
- **Query Parameters:** `telegram_id` (int64, mandatory).
- **Authentication:** `X-Internal-Secret` or `X-Internal-API-Key`.
- **Handler Implementation Logic:**
  1. Extract and validate `telegram_id`.
  2. Fetch user via `h.userRepo.FindByTelegramID(ctx, tgID)`. Return 404 `"Usuário não encontrado"` if missing.
  3. Query portfolio via `h.investService.List(ctx, u.ID)`.
  4. Query total liquid balance via `h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)`.
  5. Compute `totalNetWorth = totalLiquid.Add(portfolio.TotalInvested)`.
  6. Return 200 OK with `ListInvestmentsResponse`.
- **Response DTO:**
```go
type ListInvestmentsResponse struct {
	Investments        []investment.Investment `json:"investments"`
	TotalInvested      money.Money             `json:"total_invested"`
	TotalLiquidBalance money.Money             `json:"total_liquid_balance"`
	TotalNetWorth      money.Money             `json:"total_net_worth"`
}
```

#### Endpoint 2: `POST /internal/investments/sell`
- **Purpose:** Reduce custody or liquidate an asset position in response to `/venda` or inline callbacks.
- **Authentication:** `X-Internal-Secret` or `X-Internal-API-Key`.
- **Request DTO:**
```go
type SellInvestmentRequest struct {
	TelegramID int64           `json:"telegram_id"`
	Ticker     string          `json:"ticker"`
	Quantity   decimal.Decimal `json:"quantity"`
}
```
- **Handler Implementation Logic:**
  1. Parse JSON body. Validate `req.TelegramID != 0`, `req.Ticker != ""`, and `req.Quantity.IsPositive()`.
  2. Fetch user via `h.userRepo.FindByTelegramID(ctx, req.TelegramID)`. Return 404 if not found.
  3. Call `h.investService.Sell(ctx, u.ID, req.Ticker, req.Quantity)`.
  4. Map domain errors:
     - `ErrInvestmentNotFound` $\to$ HTTP 404 `"investimento não encontrado"`.
     - `ErrInsufficientQuantity` $\to$ HTTP 400 `"quantidade insuficiente para venda"`.
     - `ErrInvalidQuantity`, `ErrInvalidTicker` $\to$ HTTP 400 with domain error message.
  5. Fetch updated `portfolio` via `h.investService.List(ctx, u.ID)` and `totalLiquid` via `h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)`.
  6. Return 200 OK with `SellInvestmentResponse`.
- **Response DTO:**
```go
type SellInvestmentResponse struct {
	Investment         *investment.Investment `json:"investment"`
	Ticker             string                 `json:"ticker"`
	SoldQuantity       decimal.Decimal        `json:"sold_quantity"`
	RemainingQuantity  decimal.Decimal        `json:"remaining_quantity"`
	AveragePrice       money.Money            `json:"average_price"`
	IsClosed           bool                   `json:"is_closed"`
	TotalInvested      money.Money            `json:"total_invested"`
	TotalLiquidBalance money.Money            `json:"total_liquid_balance"`
	TotalNetWorth      money.Money            `json:"total_net_worth"`
}
```

### 2.3 Route Registration in `apps/api/internal/botapi/handler.go`
In `RegisterRoutes(r chi.Router)`:
```go
internal.Use(h.requireInternalSecret)
// Existing routes...
internal.Post("/internal/investments", h.handleAddInvestment)
// New routes:
internal.Get("/internal/investments", h.handleListInvestments)
internal.Post("/internal/investments/sell", h.handleSellInvestment)
```

---

## 3. Bot Client & Domain Architecture (`apps/bot`)

### 3.1 Client Layer (`apps/bot/internal/client`)

#### `types.go` Additions
```go
type InvestmentItem struct {
	ID           string    `json:"id"`
	Ticker       string    `json:"ticker"`
	Quantity     FlexFloat `json:"quantity"`
	AveragePrice float64   `json:"average_price"`
	TotalCost    float64   `json:"total_cost"`
}

type ListInvestmentsResponse struct {
	Investments        []InvestmentItem `json:"investments"`
	TotalInvested      float64          `json:"total_invested"`
	TotalLiquidBalance float64          `json:"total_liquid_balance"`
	TotalNetWorth      float64          `json:"total_net_worth"`
}

type SellInvestmentRequest struct {
	TelegramID int64   `json:"telegram_id"`
	Ticker     string  `json:"ticker"`
	Quantity   float64 `json:"quantity"`
}

type SellInvestmentResponse struct {
	Investment         InvestmentItem `json:"investment"`
	Ticker             string         `json:"ticker"`
	SoldQuantity       FlexFloat      `json:"sold_quantity"`
	RemainingQuantity  FlexFloat      `json:"remaining_quantity"`
	AveragePrice       float64        `json:"average_price"`
	IsClosed           bool           `json:"is_closed"`
	TotalInvested      float64        `json:"total_invested"`
	TotalLiquidBalance float64        `json:"total_liquid_balance"`
	TotalNetWorth      float64        `json:"total_net_worth"`
}
```

#### `api_client.go` Additions
```go
// ListInvestments obtém a lista de investimentos e patrimônio consolidado do usuário via Telegram ID.
func (c *APIClient) ListInvestments(ctx context.Context, telegramID int64) (*ListInvestmentsResponse, error) {
	url := fmt.Sprintf("%s/internal/investments?telegram_id=%d", c.baseURL, telegramID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var resp ListInvestmentsResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

// SellInvestment registra a venda ou baixa de custódia de um ativo para o usuário.
func (c *APIClient) SellInvestment(ctx context.Context, req SellInvestmentRequest) (*SellInvestmentResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar requisicao de venda: %w", err)
	}

	url := fmt.Sprintf("%s/internal/investments/sell", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var resp SellInvestmentResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}
```

---

## 4. Parser Architecture & Ticker Widening (`apps/bot/internal/bot/fsm/parsers.go`)

### 4.1 Ticker Regex Widening (`D-17`)
In `apps/bot/internal/bot/fsm/parsers.go`:
- **Current:** `tickerRegex = regexp.MustCompile("^[A-Z0-9]{4,6}$")` (restricts to 4-6 chars without symbols).
- **Target:** `tickerRegex = regexp.MustCompile("^[A-Z0-9.\\-_]{1,12}$")` (matches API regex in `investment.go`).
- **Support:** Tickers like `ALUP11`, `PETR4`, `MXRF11`, `BOVA11`, and fixed income representations: `TD-SELIC`, `CDB-INTER`, `TD-IPCA29`, `LCI-DI`.

### 4.2 Sale Command Parser Specification
Signature:
```go
func ParseSaleCommand(args string) (ticker string, qty float64, price *float64, err error)
```

#### Parsing Algorithm
1. Check if trimmed `args` is empty $\to$ return error `"Argumentos obrigatórios ausentes. Envie no formato: /venda <TICKER> <QUANTIDADE> [a <PREÇO>]"`.
2. Normalize separators:
   - `letterCommaRe.ReplaceAllString(raw, "$1 ")`
   - `commaSpaceRe.ReplaceAllString(cleaned, " ")`
3. Split into whitespace tokens.
4. Iterate tokens:
   - Skip syntactic noise: `"un"`, `"unidades"`, `"cotas"`, `"cota"`, `"a"`, `"de"`, `"r$"`, `"R$"`.
   - First attempt parsing token as a numeric value via `ParseMoney(tok)`. If successful, append to `nums []float64`.
   - If token is not numeric: uppercase the token. If it matches `tickerRegex` and `ticker == ""`, assign `ticker = upper`.
5. Validation:
   - If `ticker == ""` $\to$ return error `"Código do ativo (ticker) não informado ou inválido."`
   - If `len(nums) == 0` $\to$ return error `"Informe a quantidade a ser vendida. Exemplo: /venda TICKER QUANTIDADE [a PREÇO]"`
   - `qty = nums[0]`
   - If `qty <= 0` $\to$ return error `"Quantidade deve ser maior que zero."`
   - If `len(nums) >= 2`:
     - `priceVal := nums[1]`
     - If `priceVal < 0` $\to$ return error `"Preço de venda não pode ser negativo."`
     - `price = &priceVal`
6. Returns `(ticker, qty, price, nil)`.

---

## 5. Telegram Presentation & Interaction Design (`apps/bot/internal/bot`)

To comply with the engineering rule **"Proibido God Files"** and keep `bot.go` strictly as a router, all portfolio, contribution, and sale logic is housed in a dedicated handler file: `apps/bot/internal/bot/invest_handler.go`.

### 5.1 Command Routing & Aliases (`D-15`)

| Operation | Canonical Command | Aliases | Handler Method |
|---|---|---|---|
| Consulta de Carteira | `/carteira` | `/investimentos`, `/portfolio` | `b.handlePortfolioCommand(ctx, chatID, user.ID)` |
| Aporte / Compra | `/investimento` | `/comprar`, `/aporte` | `b.handleInvestCommand(ctx, chatID, user, text)` |
| Venda / Baixa | `/venda` | `/vender` | `b.handleSellCommand(ctx, chatID, user, text)` |

### 5.2 Educational Cards for Arguments-less Invocations (`D-16`)
When sent with no arguments:
- `/carteira` $\to$ directly renders the portfolio or empty-state card.
- `/investimento` (or `/comprar`, `/aporte`) $\to$ renders educational prompt with copyable examples.
- `/venda` (or `/vender`) $\to$ renders educational prompt with copyable examples.

### 5.3 Consolidated Portfolio Card Layout (`/carteira`, `D-01`..`D-04`, `D-12`, `D-14`)

#### Populated State
```text
📈 *Sua Carteira de Investimentos*

🔹 *ALUP11* (35,2% da carteira)
• 100 cotas • PM: R$ 42,23 • Total: R$ 4.223,00

🔹 *PETR4* (28,5% da carteira)
• 80 cotas • PM: R$ 38,50 • Total: R$ 3.080,00

🔹 *TD-SELIC* (36,3% da carteira)
• 1 cota • PM: R$ 14.500,00 • Total: R$ 14.500,00
───────────────────────────
💼 *Total Investido:* R$ 21.803,00
🏦 *Saldo em Conta:* R$ 3.197,00
🌐 *Patrimônio Líquido Total:* R$ 25.000,00
```
- **Sorting (`D-02`):** In `invest_handler.go`, items are sorted via `sort.SliceStable` by `TotalCost` descending.
- **Allocation % (`D-01`):** Calculated as `(item.TotalCost / totalInvested) * 100` and formatted as `XX,X%`.
- **Quantity formatting:** Single item formatted as `"1 cota"`; multiple items formatted as `"X cotas"` (or decimal if fractional).
- **Inline Keyboard (`D-14`):**
  - `[➕ Novo Aporte]` $\to$ callback `invest:aporte`
  - `[🔄 Atualizar]` $\to$ callback `invest:refresh` (edits the existing message in-place via `EditMessageTextConfig`).

#### Empty State (`D-04`)
```text
📊 *Sua Carteira de Investimentos*

Você ainda não possui nenhum ativo cadastrado na sua carteira!

Acompanhe suas ações da B3, Fundos Imobiliários (FIIs), ETFs e títulos de Renda Fixa diretamente por aqui com cálculo automático de Preço Médio e alocação percentual.

💡 *Como adicionar seu primeiro ativo:*
• `/investimento ALUP11 100 a 42.23`
• `/comprar MXRF11 50 a 10.50`
• `/aporte TD-SELIC 1 a 14500.00`
```
- **Inline Keyboard:** `[➕ Adicionar Ativo]` $\to$ callback `invest:aporte`.

### 5.4 Contribution Confirmation Card (`/investimento`, `INVEST-01`)
```text
✅ *Aporte registrado com sucesso!*

📈 Ativo: *ALUP11*
• Lote Adicionado: 100 cotas a R$ 42,23
• Nova Posição: 200 cotas
• Preço Médio (PM): R$ 41,50
• Total no Ativo: R$ 8.300,00

💼 *Total Investido na Carteira:* R$ 25.103,00
🌐 *Patrimônio Líquido Global:* R$ 28.300,00
```

### 5.5 Sale Confirmation Cards (`/venda`, `INVEST-03`)

#### Scenario A: Partial Sale WITH Sale Price (`D-05`, `D-06`, `D-08`, `D-10`)
Command: `/venda PETR4 30 a 41.50`
```text
✅ *Venda registrada com sucesso!*

📉 Ativo: *PETR4*
• Vendido: 30 cotas a R$ 41,50
• Total Apurado: R$ 1.245,00
• Preço Médio: R$ 38,50
• Lucro Realizado: +R$ 90,00 (+7,79%) 🟢

📊 *Custódia Remanescente:* 50 cotas (R$ 1.925,00)
```
- **Financial Calculations:**
  - $\text{Total Apurado} = \text{sellPrice} \times \text{sellQty} = 41.50 \times 30 = 1.245,00$
  - $\text{Lucro em R\$} = (\text{sellPrice} - \text{averagePrice}) \times \text{sellQty} = (41.50 - 38.50) \times 30 = 90,00$
  - $\text{Lucro em \%} = \frac{\text{sellPrice} - \text{averagePrice}}{\text{averagePrice}} \times 100 = \frac{3.00}{38.50} \times 100 = 7,79\%$
  - Prejuízo displays `-R$ X,XX (-Y,YY%) 🔴`; Neutro displays `R$ 0,00 (0,00%) ⚪`.
- **Inline Keyboard (`D-06`, `D-10`):**
  - Row 1: `[💳 Creditar R$ 1.245,00 no Saldo Líquido]` (`invest:credit:1245.00:PETR4`) | `[🛡️ Manter Apenas na Carteira]` (`invest:keep:PETR4`)
  - Row 2: `[↩️ Desfazer Venda]` (`invest:undo:PETR4:30:38.50`)

#### Scenario B: 100% Sold / Closed Position WITH Sale Price (`D-09`, `D-10`)
Command: `/venda PETR4 80 a 41.50`
```text
🏁 *Posição Encerrada!*

Você vendeu 100% das suas cotas de *PETR4*.
• Total Apurado: R$ 3.320,00
• Lucro total realizado: +R$ 240,00 (+7,79%) 🟢
• *PETR4* foi removido da sua custódia ativa.
```
- **Inline Keyboard:**
  - Row 1: `[💳 Creditar R$ 3.320,00 no Saldo Líquido]` (`invest:credit:3320.00:PETR4`) | `[🛡️ Manter Apenas na Carteira]` (`invest:keep:PETR4`)
  - Row 2: `[↩️ Desfazer Venda]` (`invest:undo:PETR4:80:38.50`)

#### Scenario C: Sale WITHOUT Sale Price (`D-05`)
Command: `/venda PETR4 30`
```text
✅ *Baixa de custódia registrada com sucesso!*

📉 Ativo: *PETR4*
• Baixa: 30 cotas
• Preço Médio: R$ 38,50

📊 *Custódia Remanescente:* 50 cotas (R$ 1.925,00)

💡 *Dica:* Para calcular lucro/prejuízo e creditar no saldo, informe o preço: `/venda PETR4 30 a 41.50`
```
- **Inline Keyboard:** `[↩️ Desfazer Venda]` (`invest:undo:PETR4:30:38.50`).

#### Scenario D: Excess Custody Attempt (`D-07`)
Command: `/venda PETR4 100` (User only owns 80 cotas)
- API returns `400 Bad Request` with `"quantidade insuficiente para venda"`.
- The bot retrieves user's current holdings via `ListInvestments` to discover current custody.
```text
⚠️ *Quantidade insuficiente para venda!*

Você possui *80 cotas* de *PETR4* em carteira, mas tentou vender *100 cotas*.

💡 Deseja liquidar toda a sua posição atual?
```
- **Inline Keyboard:**
  - Row 1: `[Vender Todas as 80]` (`invest:sell_all:PETR4:80:[price]`) | `[❌ Cancelar]` (`invest:cancel`)

### 5.6 Callback Handlers (`invest:*`)

1. **`invest:aporte`**:
   Sends instructional message guiding the user to run `/investimento <TICKER> <QTD> [a] <PREÇO>`.
2. **`invest:refresh`**:
   Calls `b.client.ListInvestments(ctx, user.ID)`. Re-renders the card and updates the existing message with `tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, cardText, keyboard)`.
3. **`invest:credit:<amount>:<ticker>`**:
   Calls `b.client.RegisterIncome(ctx, client.IncomeRequest{ TelegramID: user.ID, Amount: amount, Description: "Venda de " + ticker, CategoryName: "Investimentos" })`.
   Updates message confirming the credit and showing the new increased S2S and liquid balance.
4. **`invest:keep:<ticker>`**:
   Acknowledges with friendly message stating proceeds remain in the broker/portfolio without affecting the daily S2S.
5. **`invest:undo:<ticker>:<qty>:<price>`**:
   Calls `b.client.AddInvestment(ctx, client.AddInvestmentRequest{ TelegramID: user.ID, Ticker: ticker, Quantity: qty, Price: price })`.
   Restores the custody and PM immediately, and updates the message confirming the reversal.
6. **`invest:sell_all:<ticker>:<qty>:[price]`**:
   Executes sale of the full available position, triggering Scenario B.
7. **`invest:cancel`**:
   Dismisses the prompt with `"❌ Operação cancelada."`

---

## 6. Testing Strategy & Troféu de Testes

### 6.1 Troféu de Testes Compliance
- **Unit Tests:**
  - Algorithm testing for parsers (`parsers_test.go`): Ticker validation, flexible tokenization, sale command extraction with and without price.
  - Bot interaction testing (`bot_invest_test.go`): Mock HTTP server for `client.APIClient` and `mockSender` for Telegram Chattables. Verifies card formatting, keyboard construction, and callback flows.
- **Integration Tests:**
  - Real PostgreSQL 18 container via Testcontainers (`apps/api/tests/integration/internal_bot_api_investment_test.go`).
  - No database mocks (`sqlmock` is forbidden).
  - Tests verify transactions, concurrency locking (`FOR UPDATE`), database deletes on zero position, and clean cycle initiation on repurchase.

### 6.2 Test Inventory

| Test File | Test Name | Target Behavior |
|---|---|---|
| `apps/bot/internal/bot/fsm/parsers_test.go` | `TestParseInvestment_FixedIncomeAndEquities` | Tickers `TD-SELIC`, `CDB-INTER`, `TD-IPCA29` parse with correct qty and price |
| `apps/bot/internal/bot/fsm/parsers_test.go` | `TestParseSaleCommand_SuccessCases` | Parse `/venda` with price, without price, commas, units, prefixes |
| `apps/bot/internal/bot/fsm/parsers_test.go` | `TestParseSaleCommand_ValidationErrors` | Empty, missing quantity, zero qty, negative qty/price |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestInvestCommand_HelpGuide` | `/investimento` without args shows educational guide |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestInvestCommand_SuccessFlow_Aliases` | `/investimento`, `/comprar`, `/aporte` invoke API and reply card |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestPortfolioCommand_Empty` | `/carteira` with no assets returns encouraging empty card |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestPortfolioCommand_Populated_SortingAndAllocation` | Assets sorted by R$ volume DESC, allocation % formatted, footer metrics |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestPortfolioCallback_RefreshAndAporte` | Callbacks `invest:refresh` and `invest:aporte` execute properly |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCommand_HelpGuide` | `/venda` without args shows educational guide |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCommand_PartialSale_WithProfitAndLoss` | Calculates profit/loss in R$ and %, renders credit and undo buttons |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCommand_ClosedPosition` | 100% sale renders position closed card and undo button |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCommand_WithoutPrice` | Sale without price updates custody without financial P&L |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCommand_InsufficientQuantity_Shortcut` | Excess sale renders error message and `[Vender Todas as X]` button |
| `apps/bot/internal/bot/bot_invest_test.go` | `TestSellCallback_CreditAndUndo` | `invest:credit` registers income; `invest:undo` restores position |
| `apps/api/tests/integration/internal_bot_api_investment_test.go` | `TestInternalBotAPI_ListInvestments_EmptyAndPopulated` | Verifies `GET /internal/investments` with DB state |
| `apps/api/tests/integration/internal_bot_api_investment_test.go` | `TestInternalBotAPI_SellInvestment_PartialAndZero` | Verifies `POST /internal/investments/sell` updates DB and deletes row on zero |
| `apps/api/tests/integration/internal_bot_api_investment_test.go` | `TestInternalBotAPI_SellInvestment_Errors` | 404 on unknown asset, 400 on insufficient quantity |
| `apps/api/tests/integration/internal_bot_api_investment_test.go` | `TestInternalBotAPI_RepurchaseStartsCleanCycle` | Selling 100% then buying again creates fresh PM equal to purchase price (`D-11`) |

---

## Validation Architecture

The Nyquist validation framework guarantees that every requirement is backed by concrete, executable, deterministic tests:

```text
Nyquist Validation Grid:
┌───────────┬──────────────────────────────────────────┬────────────────────────────────────────────────────────┐
│ Req ID    │ Verification Target                      │ Verification Test / Command                            │
├───────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ INVEST-01 │ Aporte via Telegram com recálculo de PM  │ go test -v ./internal/bot -run TestInvestCommand       │
│           │ e suporte a Renda Fixa                   │ go test -v ./internal/bot/fsm -run TestParseInvestment │
├───────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ INVEST-02 │ Consulta de carteira consolidada, cards, │ go test -v ./internal/bot -run TestPortfolio           │
│           │ ordenação por R$, alocação % e rodapé    │ go test -v ./tests/integration -run ListInvestments    │
├───────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ INVEST-03 │ Venda com baixa de custódia, apuração    │ go test -v ./internal/bot -run TestSell                │
│           │ de P&L, posição zerada, crédito e undo   │ go test -v ./tests/integration -run SellInvestment     │
└───────────┴──────────────────────────────────────────┴────────────────────────────────────────────────────────┘
```

### Verification Commands
```bash
# Executar testes unitários do bot e parsers
cd apps/bot && go test -v ./internal/bot/... ./internal/bot/fsm/...

# Executar testes de integração reais contra Postgres 18 via Testcontainers
cd apps/api && go test -v ./tests/integration -run 'TestInternalBotAPI_.*Investment'

# Validação global da suíte
make test-bot
make test-api
```

---

## 8. Risks, Edge Cases & Defenses

| Risk / Edge Case | Likelihood | Impact | Defensive Strategy |
|---|---|---|---|
| **Concorrência e Venda Duplicada** | Baixa | Alta | `repository.Sell` e `AddLot` utilizam `SELECT ... FOR UPDATE` no PostgreSQL 18 para serializar transações no mesmo ativo. |
| **Erros de Ponto Flutuante em Moeda** | Média | Alta | A API utiliza estritamente `money.Money` e `decimal.Decimal` com `RoundBank(HalfEven)`. O bot utiliza `formatMoneyBR` e `FlexFloat`. |
| **Recompra após Zeramento de Ativo** | Média | Média | Ao zerar, a linha é deletada de `tb_investments`. Nova compra inicia novo ciclo limpo onde PM = preço da compra (`D-11`). |
| **Tentativa de Vender Quantidade Superior à Custódia** | Alta | Baixa | A API retorna `ErrInsufficientQuantity` (400). O bot intercepta, busca a custódia atual e fornece o botão de 1 clique `[Vender Todas as X]`. |
| **Erro de Digitação na Venda** | Média | Média | Botão `[↩️ Desfazer Venda]` incluído em toda confirmação de venda para recomprar imediatamente a quantidade vendida pelo Preço Médio histórico. |
| **Crédito Indesejado no S2S** | Baixa | Alta | O crédito no saldo bancário nunca é automático; exige clique explícito em `[💳 Creditar R$ X no Saldo Líquido]`. |
| **Mensagens em Outro Idioma** | Baixa | Média | Todas as mensagens de erro, descrições e badges são 100% em Português (pt-BR) conforme diretrizes do `AGENTS.md`. |
| **Crescimento de God Files** | Baixa | Média | Toda a lógica de investimento no bot é concentrada em `invest_handler.go`, mantendo `bot.go` limpo e modular. |

---

## 9. Conclusion & Planning Readiness

Phase 3 is fully specified with clear boundaries, reusability of existing domain code, unambiguous DTO contracts, robust parsing rules, and comprehensive test suites covering both unit and Testcontainers integration layers. 

The phase is ready for detailed planning (`PLAN.md`).
