package bot_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type investAPIMockServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	investments   []client.InvestmentItem
	totalLiquid   float64
	lastAddReq    *client.AddInvestmentRequest
	lastSellReq   *client.SellInvestmentRequest
	lastIncomeReq *client.IncomeRequest
}

func newInvestAPIMockServer() *investAPIMockServer {
	mock := &investAPIMockServer{
		investments: []client.InvestmentItem{},
		totalLiquid: 5000.00,
	}

	mux := http.NewServeMux()

	// GET /internal/investments
	mux.HandleFunc("/internal/investments", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		if r.Method == http.MethodGet {
			totalInvested := 0.0
			for _, inv := range mock.investments {
				totalInvested += inv.TotalCost
			}
			resp := client.ListInvestmentsResponse{
				Investments:        mock.investments,
				TotalInvested:      totalInvested,
				TotalLiquidBalance: mock.totalLiquid,
				TotalNetWorth:      totalInvested + mock.totalLiquid,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodPost {
			var req client.AddInvestmentRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mock.lastAddReq = &req

			// Simula recálculo de PM e quantidade
			var found bool
			for i, inv := range mock.investments {
				if inv.Ticker == req.Ticker {
					found = true
					newQty := float64(inv.Quantity) + req.Quantity
					newTotal := inv.TotalCost + (req.Quantity * req.Price)
					newPM := newTotal / newQty
					mock.investments[i].Quantity = client.FlexFloat(newQty)
					mock.investments[i].AveragePrice = newPM
					mock.investments[i].TotalCost = newTotal
					break
				}
			}
			if !found {
				newInv := client.InvestmentItem{
					ID:           "inv-" + req.Ticker,
					Ticker:       req.Ticker,
					Quantity:     client.FlexFloat(req.Quantity),
					AveragePrice: req.Price,
					TotalCost:    req.Quantity * req.Price,
				}
				mock.investments = append(mock.investments, newInv)
			}

			totalInvested := 0.0
			for _, inv := range mock.investments {
				totalInvested += inv.TotalCost
			}

			var matchedInv client.InvestmentItem
			for _, inv := range mock.investments {
				if inv.Ticker == req.Ticker {
					matchedInv = inv
					break
				}
			}

			resp := client.AddInvestmentResponse{
				Investment:    matchedInv,
				TotalInvested: totalInvested,
				TotalNetWorth: totalInvested + mock.totalLiquid,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// POST /internal/investments/sell
	mux.HandleFunc("/internal/investments/sell", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		var req client.SellInvestmentRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.lastSellReq = &req

		// Localiza ativo
		var index = -1
		for i, inv := range mock.investments {
			if inv.Ticker == req.Ticker {
				index = i
				break
			}
		}

		if index == -1 {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "investimento não encontrado"})
			return
		}

		inv := mock.investments[index]
		currentQty := float64(inv.Quantity)
		if req.Quantity > currentQty {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "quantidade insuficiente para venda"})
			return
		}

		remQty := currentQty - req.Quantity
		isClosed := remQty <= 0

		if isClosed {
			mock.investments = append(mock.investments[:index], mock.investments[index+1:]...)
		} else {
			mock.investments[index].Quantity = client.FlexFloat(remQty)
			mock.investments[index].TotalCost = remQty * inv.AveragePrice
		}

		totalInvested := 0.0
		for _, item := range mock.investments {
			totalInvested += item.TotalCost
		}

		resp := client.SellInvestmentResponse{
			Ticker:             inv.Ticker,
			SoldQuantity:       client.FlexFloat(req.Quantity),
			RemainingQuantity:  client.FlexFloat(remQty),
			AveragePrice:       inv.AveragePrice,
			IsClosed:           isClosed,
			TotalInvested:      totalInvested,
			TotalLiquidBalance: mock.totalLiquid,
			TotalNetWorth:      totalInvested + mock.totalLiquid,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /internal/transactions/income
	mux.HandleFunc("/internal/transactions/income", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		var req client.IncomeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.lastIncomeReq = &req
		mock.totalLiquid += req.Amount

		resp := client.IncomeResponse{
			CategoryName:       req.CategoryName,
			PreviousS2S:        50.00,
			NewS2S:             115.50,
			HealthStatus:       "HEALTHY",
			DaysRemaining:      20,
			TotalLiquidBalance: mock.totalLiquid,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func setupInvestBot(apiMock *investAPIMockServer) (*bot.Bot, *mockSender) {
	sender := &mockSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiMock.server.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, &config.Config{})
	b.SetSender(sender)
	return b, sender
}

func TestInvestCommand(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// 1. /investimento sem argumentos exibe card educativo
	updateHelp := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/investimento",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateHelp)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Registro de Aporte de Investimento")
	assert.Contains(t, lastMsg.Text, "/investimento <TICKER> <QUANTIDADE> [a] <PREÇO>")
	assert.Contains(t, lastMsg.Text, "• `/investimento ALUP11 100 42.23`")

	// 2. /aporte sem argumentos (alias)
	updateAporte := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/aporte",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateAporte)
	assert.Contains(t, sender.LastText(), "Registro de Aporte de Investimento")

	// 3. /comprar sem argumentos (alias)
	updateComprar := &tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "/comprar",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateComprar)
	assert.Contains(t, sender.LastText(), "Registro de Aporte de Investimento")

	// 4. Executa aporte /investimento ALUP11 100 a 42.23 (INVEST-01)
	updateBuy := &tgbotapi.Update{
		UpdateID: 4,
		Message: &tgbotapi.Message{
			Text: "/investimento ALUP11 100 a 42.23",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateBuy)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastAddReq)
	assert.Equal(t, "ALUP11", mockAPI.lastAddReq.Ticker)
	assert.Equal(t, 100.0, mockAPI.lastAddReq.Quantity)
	assert.Equal(t, 42.23, mockAPI.lastAddReq.Price)
	mockAPI.mu.Unlock()

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Aporte registrado com sucesso!")
	assert.Contains(t, lastMsg.Text, "`ALUP11`")
	assert.Contains(t, lastMsg.Text, "100 cotas")
	assert.Contains(t, lastMsg.Text, "42,23")
	assert.Contains(t, lastMsg.Text, "Total na Carteira")
}

func TestPortfolioCommand_Empty(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// 1. /carteira vazia
	updateCarteira := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/carteira",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateCarteira)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Sua Carteira de Ativos está vazia!")
	assert.Contains(t, lastMsg.Text, "Você ainda não possui ações ou títulos cadastrados")
	require.NotNil(t, lastMsg.ReplyMarkup)

	// Valida botão inline [➕ Adicionar Ativo]
	inlineMarkup, ok := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, ok)
	require.Len(t, inlineMarkup.InlineKeyboard, 1)
	assert.Equal(t, "➕ Adicionar Ativo", inlineMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "invest:aporte", *inlineMarkup.InlineKeyboard[0][0].CallbackData)

	// 2. Aliases /investimentos e /portfolio produzem mesmo efeito (D-15)
	updateInvestimentos := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/investimentos",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updateInvestimentos)
	assert.Contains(t, sender.LastText(), "Sua Carteira de Ativos está vazia!")

	updatePortfolio := &tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "/portfolio",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, updatePortfolio)
	assert.Contains(t, sender.LastText(), "Sua Carteira de Ativos está vazia!")
}

func TestPortfolioCommand_Populated_SortingAndAllocation(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	// Popula carteira com 3 ativos com volumes distintos:
	// ALUP11: 100 cotas a R$ 45,00 -> Total R$ 4.500,00 (~22.0%)
	// PETR4: 50 cotas a R$ 30,00 -> Total R$ 1.500,00 (~7.3%)
	// TD-SELIC: 1 cota a R$ 14.500,00 -> Total R$ 14.500,00 (~70.7%)
	// Total Investido: R$ 20.500,00 | Total Líquido: R$ 5.000,00 | Total Net Worth: R$ 25.500,00
	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "ALUP11", Quantity: 100, AveragePrice: 45.00, TotalCost: 4500.00},
		{ID: "2", Ticker: "PETR4", Quantity: 50, AveragePrice: 30.00, TotalCost: 1500.00},
		{ID: "3", Ticker: "TD-SELIC", Quantity: 1, AveragePrice: 14500.00, TotalCost: 14500.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/carteira",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	text := lastMsg.Text

	assert.Contains(t, text, "Sua Carteira de Investimentos")

	// Valida ordenação por maior volume financeiro investido (R$) decrescente (D-02):
	// TD-SELIC (14.500) deve vir antes de ALUP11 (4.500), que deve vir antes de PETR4 (1.500)
	idxSelic := indexOf(text, "TD-SELIC")
	idxAlup := indexOf(text, "ALUP11")
	idxPetr := indexOf(text, "PETR4")
	require.True(t, idxSelic < idxAlup, "TD-SELIC deve aparecer antes de ALUP11")
	require.True(t, idxAlup < idxPetr, "ALUP11 deve aparecer antes de PETR4")

	// Valida percentuais de alocação (D-01)
	assert.Contains(t, text, "70.7% da carteira")
	assert.Contains(t, text, "22.0% da carteira")
	assert.Contains(t, text, "7.3% da carteira")

	// Valida rodapé consolidado (D-03)
	assert.Contains(t, text, "Total Investido:* R$ 20500,00")
	assert.Contains(t, text, "Saldo em Conta:* R$ 5000,00")
	assert.Contains(t, text, "Patrimônio Líquido Total:* R$ 25500,00")

	// Valida botões inline [➕ Novo Aporte] e [🔄 Atualizar] (D-14)
	inlineMarkup, ok := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, ok)
	require.Len(t, inlineMarkup.InlineKeyboard, 1)
	assert.Equal(t, "➕ Novo Aporte", inlineMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "invest:aporte", *inlineMarkup.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "🔄 Atualizar", inlineMarkup.InlineKeyboard[0][1].Text)
	assert.Equal(t, "invest:refresh", *inlineMarkup.InlineKeyboard[0][1].CallbackData)
}

func TestSellCommand_PartialSale_WithProfitAndLoss(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	// Usuário possui 80 PETR4 com PM de R$ 38,50 (Total R$ 3.080,00)
	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "PETR4", Quantity: 80, AveragePrice: 38.50, TotalCost: 3080.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// Venda parcial: 30 cotas a R$ 41,50
	// Total apurado = 30 * 41.50 = R$ 1.245,00
	// Lucro = (41.50 - 38.50) * 30 = +R$ 90,00 (+7,79%)
	// Custódia remanescente = 50 cotas
	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/venda PETR4 30 a 41.50",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	text := lastMsg.Text

	assert.Contains(t, text, "Venda registrada com sucesso!")
	assert.Contains(t, text, "`PETR4`")
	assert.Contains(t, text, "30 cotas")
	assert.Contains(t, text, "R$ 41,50")
	assert.Contains(t, text, "R$ 38,50")
	assert.Contains(t, text, "Total Apurado:* R$ 1245,00")
	assert.Contains(t, text, "🟢 *Lucro Realizado:* +R$ 90,00 (+7.79%)")
	assert.Contains(t, text, "Custódia Remanescente:* 50 cotas")

	// Teclado inline com crédito e desfazer
	inlineMarkup, ok := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, ok)
	assert.Contains(t, inlineMarkup.InlineKeyboard[0][0].Text, "Creditar R$ 1245,00 no Saldo Líquido")
	assert.Equal(t, "invest:credit:1245.00:PETR4", *inlineMarkup.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "🛡️ Manter Apenas na Carteira", inlineMarkup.InlineKeyboard[0][1].Text)
	assert.Equal(t, "invest:keep:PETR4", *inlineMarkup.InlineKeyboard[0][1].CallbackData)
	assert.Equal(t, "↩️ Desfazer Venda", inlineMarkup.InlineKeyboard[1][0].Text)
	assert.Equal(t, "invest:undo:PETR4:30:38.50", *inlineMarkup.InlineKeyboard[1][0].CallbackData)
}

func TestSellCommand_ClosedPosition(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	// Usuário possui 80 PETR4 com PM de R$ 38,50
	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "PETR4", Quantity: 80, AveragePrice: 38.50, TotalCost: 3080.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// Venda de 100% da posição (80 cotas) a R$ 41,50 (D-09)
	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/venda PETR4 80 a 41.50",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	text := lastMsg.Text

	assert.Contains(t, text, "🏁 *Posição Encerrada!*")
	assert.Contains(t, text, "Todas as cotas de *PETR4* foram vendidas.")
	assert.Contains(t, text, "Total Apurado:* R$ 3320,00")
	assert.NotContains(t, text, "Custódia Remanescente") // posição zerada, sem remanescente
}

func TestSellCommand_WithoutPrice(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "PETR4", Quantity: 80, AveragePrice: 38.50, TotalCost: 3080.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// Venda sem preço (/venda PETR4 30) (D-05, D-10)
	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/venda PETR4 30",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	text := lastMsg.Text

	assert.Contains(t, text, "Baixa de custódia registrada!")
	assert.Contains(t, text, "Quantidade Baixada:* 30 cotas")
	assert.Contains(t, text, "Custódia Remanescente:* 50 cotas")
	assert.NotContains(t, text, "Lucro Realizado") // sem preço, não apura P&L
	assert.Contains(t, text, "Dica: Para apurar lucro ou prejuízo da operação, envie o preço de venda")

	// Permite desfazer
	inlineMarkup, ok := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, ok)
	assert.Equal(t, "↩️ Desfazer Venda", inlineMarkup.InlineKeyboard[0][0].Text)
}

func TestSellCommand_InsufficientQuantity_Shortcut(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	// Usuário tem apenas 80 cotas
	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "PETR4", Quantity: 80, AveragePrice: 38.50, TotalCost: 3080.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// Tenta vender 100 cotas (excesso de custódia) (D-07)
	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/venda PETR4 100",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	b.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	text := lastMsg.Text

	assert.Contains(t, text, "Custódia insuficiente para venda de PETR4!")
	assert.Contains(t, text, "Você solicitou a venda de 100 cotas, mas possui apenas *80 cotas* em custódia.")

	// Valida botão inteligente de atalho para vender todas as 80
	inlineMarkup, ok := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, ok)
	assert.Equal(t, "Vender Todas as 80", inlineMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "invest:sell_all:PETR4:80:0", *inlineMarkup.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "❌ Cancelar", inlineMarkup.InlineKeyboard[0][1].Text)
}

func TestInvestCallbacks(t *testing.T) {
	mockAPI := newInvestAPIMockServer()
	defer mockAPI.server.Close()

	mockAPI.mu.Lock()
	mockAPI.investments = []client.InvestmentItem{
		{ID: "1", Ticker: "PETR4", Quantity: 50, AveragePrice: 38.50, TotalCost: 1925.00},
	}
	mockAPI.mu.Unlock()

	b, sender := setupInvestBot(mockAPI)
	ctx := context.Background()

	// 1. Callback invest:refresh (in-place edit) (D-14)
	cbRefresh := &tgbotapi.Update{
		UpdateID: 1,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-1",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:refresh",
			Message: &tgbotapi.Message{
				MessageID: 42,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbRefresh)
	editMsg, ok := sender.LastEditMessage()
	require.True(t, ok)
	assert.Contains(t, editMsg.Text, "Sua Carteira de Investimentos")
	assert.Equal(t, 42, editMsg.MessageID)

	// 2. Callback invest:credit:1245.00:PETR4 (D-06)
	cbCredit := &tgbotapi.Update{
		UpdateID: 2,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-2",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:credit:1245.00:PETR4",
			Message: &tgbotapi.Message{
				MessageID: 43,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbCredit)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastIncomeReq)
	assert.Equal(t, 1245.00, mockAPI.lastIncomeReq.Amount)
	assert.Equal(t, "Venda de PETR4", mockAPI.lastIncomeReq.Description)
	assert.Equal(t, "Investimentos", mockAPI.lastIncomeReq.CategoryName)
	mockAPI.mu.Unlock()

	editMsg, ok = sender.LastEditMessage()
	require.True(t, ok)
	assert.Contains(t, editMsg.Text, "Crédito confirmado!")
	assert.Contains(t, editMsg.Text, "R$ 1245,00")

	// 3. Callback invest:keep:PETR4 (D-06)
	cbKeep := &tgbotapi.Update{
		UpdateID: 3,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-3",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:keep:PETR4",
			Message: &tgbotapi.Message{
				MessageID: 44,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbKeep)
	editMsg, ok = sender.LastEditMessage()
	require.True(t, ok)
	assert.Contains(t, editMsg.Text, "Saldo mantido na corretora!")

	// 4. Callback invest:undo:PETR4:30:38.50 (D-10)
	cbUndo := &tgbotapi.Update{
		UpdateID: 4,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-4",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:undo:PETR4:30:38.50",
			Message: &tgbotapi.Message{
				MessageID: 45,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbUndo)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastAddReq)
	assert.Equal(t, "PETR4", mockAPI.lastAddReq.Ticker)
	assert.Equal(t, 30.0, mockAPI.lastAddReq.Quantity)
	assert.Equal(t, 38.50, mockAPI.lastAddReq.Price)
	mockAPI.mu.Unlock()

	editMsg, ok = sender.LastEditMessage()
	require.True(t, ok)
	assert.Contains(t, editMsg.Text, "Venda desfeita com sucesso!")

	// 5. Callback invest:sell_all:PETR4:50:41.50
	cbSellAll := &tgbotapi.Update{
		UpdateID: 5,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-5",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:sell_all:PETR4:50:41.50",
			Message: &tgbotapi.Message{
				MessageID: 46,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbSellAll)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastSellReq)
	assert.Equal(t, "PETR4", mockAPI.lastSellReq.Ticker)
	assert.Equal(t, 50.0, mockAPI.lastSellReq.Quantity)
	mockAPI.mu.Unlock()

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Venda registrada com sucesso!")

	// 6. Callback invest:cancel (D-07)
	cbCancel := &tgbotapi.Update{
		UpdateID: 6,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-6",
			From: &tgbotapi.User{ID: 12345, FirstName: "Investidor"},
			Data: "invest:cancel",
			Message: &tgbotapi.Message{
				MessageID: 47,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	b.ProcessUpdate(ctx, cbCancel)
	editMsg, ok = sender.LastEditMessage()
	require.True(t, ok)
	assert.Contains(t, editMsg.Text, "Operação de venda cancelada")
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
