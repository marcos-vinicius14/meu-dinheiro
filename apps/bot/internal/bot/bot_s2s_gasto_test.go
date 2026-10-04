package bot_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type botAPIServerMock struct {
	server       *httptest.Server
	userContext  client.UserFinancialContext
	quickExpense client.QuickExpenseResponse
	lastQuickReq *client.QuickExpenseRequest
	mu           sync.Mutex
}

func newBotAPIServerMock() *botAPIServerMock {
	mock := &botAPIServerMock{}

	ctxJSON := `{
		"user": {
			"id": "u-12345",
			"telegram_id": 12345,
			"first_name": "Marcos",
			"cycle_start_day": 5
		},
		"total_liquid_balance": 4500.00,
		"cycle": {
			"start_date": "2026-10-05",
			"end_date": "2026-11-04",
			"days_remaining": 22,
			"s2s_today": 78.50,
			"health_status": "HEALTHY"
		},
		"emergency_fund": {
			"monthly_essential_cost": 2000.00,
			"target": 12000.00,
			"months": 6,
			"months_covered": 2.25,
			"progress_percent": 37.5
		},
		"total_invested": 0.0,
		"total_net_worth": 4500.0
	}`
	_ = json.Unmarshal([]byte(ctxJSON), &mock.userContext)

	mock.quickExpense = client.QuickExpenseResponse{
		CategoryName:  "Alimentação",
		PreviousS2S:   85.00,
		NewS2S:        78.50,
		HealthStatus:  "HEALTHY",
		DaysRemaining: 22,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/users/context-by-telegram", func(w http.ResponseWriter, r *http.Request) {
		tgID := r.URL.Query().Get("telegram_id")
		if tgID == "12345" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mock.userContext)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário não encontrado"})
	})

	mux.HandleFunc("/internal/transactions/quick-expense", func(w http.ResponseWriter, r *http.Request) {
		var req client.QuickExpenseRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.mu.Lock()
		mock.lastQuickReq = &req
		mock.mu.Unlock()

		resp := mock.quickExpense
		resp.Transaction.Description = req.Description
		resp.Transaction.Amount = req.Amount
		resp.CategoryName = req.CategoryName

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func setupS2SGastoBot(apiMock *botAPIServerMock) (*bot.Bot, *mockSender) {
	sender := &mockSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiMock.server.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, &config.Config{})
	b.SetSender(sender)
	return b, sender
}

func TestS2S_CommandAndPersistentButton(t *testing.T) {
	mockAPI := newBotAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupS2SGastoBot(mockAPI)
	ctx := context.Background()

	// Cenário 1a: Envio do comando /s2s por usuário existente
	updateCmd := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/s2s",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}

	testBot.ProcessUpdate(ctx, updateCmd)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Saldo Seguro Diário (S2S)")
	assert.Contains(t, lastMsg.Text, "R$ 78,50/dia")
	assert.Contains(t, lastMsg.Text, "🟢 SAUDÁVEL")
	assert.Contains(t, lastMsg.Text, "22 dias")

	// Cenário 1b: Toque no botão persistente "💰 S2S Hoje"
	updateBtn := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: bot.ButtonS2SToday,
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}

	testBot.ProcessUpdate(ctx, updateBtn)

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Saldo Seguro Diário (S2S)")
	assert.Contains(t, lastMsg.Text, "R$ 78,50/dia")

	// Cenário 1c: Usuário inexistente
	updateUnknown := &tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "/s2s",
			From: &tgbotapi.User{ID: 99999, FirstName: "Desconhecido"},
			Chat: &tgbotapi.Chat{ID: 200},
		},
	}

	testBot.ProcessUpdate(ctx, updateUnknown)

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "não possui cadastro")
	assert.Contains(t, lastMsg.Text, "/start")
}

func TestGasto_WithoutArguments_ShowsGuide(t *testing.T) {
	mockAPI := newBotAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupS2SGastoBot(mockAPI)
	ctx := context.Background()

	// Cenário 2a: /gasto sem argumentos
	updateCmd := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/gasto",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}

	testBot.ProcessUpdate(ctx, updateCmd)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Lançamento Rápido de Gasto")
	assert.Contains(t, lastMsg.Text, "/gasto <valor> <descrição>")
	assert.Contains(t, lastMsg.Text, "Exemplos práticos")

	// Cenário 2b: Toque no botão persistente "💸 Lançar Gasto"
	updateBtn := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: bot.ButtonQuickExpense,
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}

	testBot.ProcessUpdate(ctx, updateBtn)

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Lançamento Rápido de Gasto")
}

func TestGasto_WithArguments_ShowsCategoryInlineButtons_AndExecutesQuickExpense(t *testing.T) {
	mockAPI := newBotAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupS2SGastoBot(mockAPI)
	ctx := context.Background()

	// 1. Envia /gasto 35.00 Almoço executivo
	updateGasto := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/gasto 35.00 Almoço executivo",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}

	testBot.ProcessUpdate(ctx, updateGasto)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Selecione a categoria para esta despesa")
	assert.Contains(t, lastMsg.Text, "35,00")
	assert.Contains(t, lastMsg.Text, "Almoço executivo")

	// Valida que o teclado inline de categorias foi anexado
	inlineMarkup, isInline := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, isInline, "Mensagem deve conter botões inline de categoria")
	require.NotEmpty(t, inlineMarkup.InlineKeyboard)

	var foundAlimentacao bool
	for _, row := range inlineMarkup.InlineKeyboard {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Alimentação") && btn.CallbackData != nil && strings.HasPrefix(*btn.CallbackData, "gasto_cat:") {
				foundAlimentacao = true
			}
		}
	}
	assert.True(t, foundAlimentacao, "Deve conter botão com a categoria Alimentação")

	// 2. Simula o clique no botão inline da categoria Alimentação
	updateCallback := &tgbotapi.Update{
		UpdateID: 2,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-101",
			Data: "gasto_cat:Alimentação",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Message: &tgbotapi.Message{
				MessageID: 1,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}

	testBot.ProcessUpdate(ctx, updateCallback)

	// Valida que a chamada ao client da API foi feita com os dados corretos
	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastQuickReq)
	assert.Equal(t, int64(12345), mockAPI.lastQuickReq.TelegramID)
	assert.Equal(t, 35.00, mockAPI.lastQuickReq.Amount)
	assert.Equal(t, "Almoço executivo", mockAPI.lastQuickReq.Description)
	assert.Equal(t, "Alimentação", mockAPI.lastQuickReq.CategoryName)
	mockAPI.mu.Unlock()

	// Valida que a resposta contém o comparativo Antes vs Depois
	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Gasto de R$ 35,00 registrado em Alimentação!")
	assert.Contains(t, lastMsg.Text, "S2S Anterior:")
	assert.Contains(t, lastMsg.Text, "R$ 85,00/dia")
	assert.Contains(t, lastMsg.Text, "Novo S2S:")
	assert.Contains(t, lastMsg.Text, "R$ 78,50/dia")
	assert.Contains(t, lastMsg.Text, "-R$ 6,50/dia")
	assert.Contains(t, lastMsg.Text, "22 dias")
}

func TestGasto_CancelCallback(t *testing.T) {
	mockAPI := newBotAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupS2SGastoBot(mockAPI)
	ctx := context.Background()

	// Envia /gasto
	updateGasto := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/gasto 50 Uber",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateGasto)

	// Simula cancelamento
	updateCancel := &tgbotapi.Update{
		UpdateID: 2,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-cancel",
			Data: "gasto_cancel",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Message: &tgbotapi.Message{
				MessageID: 1,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	testBot.ProcessUpdate(ctx, updateCancel)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Lançamento cancelado")
}

func TestPersistentMenuKeyboard_Structure(t *testing.T) {
	kb := bot.PersistentMenuKeyboard()
	require.Len(t, kb.Keyboard, 2)
	assert.Len(t, kb.Keyboard[0], 2)
	assert.Len(t, kb.Keyboard[1], 2)

	assert.Equal(t, bot.ButtonS2SToday, kb.Keyboard[0][0].Text)
	assert.Equal(t, bot.ButtonQuickExpense, kb.Keyboard[0][1].Text)
	assert.Equal(t, bot.ButtonSimulate, kb.Keyboard[1][0].Text)
	assert.Equal(t, bot.ButtonCheckin, kb.Keyboard[1][1].Text)
	assert.True(t, kb.ResizeKeyboard)
	assert.False(t, kb.OneTimeKeyboard)
}
