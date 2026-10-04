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

type simAPIServerMock struct {
	server       *httptest.Server
	simulateResp client.SimulationResponse
	lastSimReq   *client.SimulationRequest
	lastQuickReq *client.QuickExpenseRequest
	quickResp    client.QuickExpenseResponse
	mu           sync.Mutex
}

func newSimAPIServerMock() *simAPIServerMock {
	mock := &simAPIServerMock{
		simulateResp: client.SimulationResponse{
			CurrentCycleS2SBefore:     100.00,
			CurrentCycleS2SAfter:      80.00,
			CriticalCycleNumber:       4,
			CriticalCycleS2S:          45.00,
			CriticalCycleHealthStatus: "ATTENTION",
			DeficitRiskAlert:          false,
			RecommendationMessage:     "",
		},
		quickResp: client.QuickExpenseResponse{
			CategoryName:  "Outros",
			PreviousS2S:   100.00,
			NewS2S:        80.00,
			HealthStatus:  "HEALTHY",
			DaysRemaining: 20,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/transactions/simulations", func(w http.ResponseWriter, r *http.Request) {
		var req client.SimulationRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.mu.Lock()
		mock.lastSimReq = &req
		resp := mock.simulateResp
		mock.mu.Unlock()

		if req.TelegramID == 99999 {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário não encontrado"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/internal/transactions/quick-expense", func(w http.ResponseWriter, r *http.Request) {
		var req client.QuickExpenseRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.mu.Lock()
		mock.lastQuickReq = &req
		resp := mock.quickResp
		mock.mu.Unlock()

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

func setupSimularBot(apiMock *simAPIServerMock) (*bot.Bot, *mockSender) {
	sender := &mockSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiMock.server.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, &config.Config{})
	b.SetSender(sender)
	return b, sender
}

func TestSimular_HelpGuide(t *testing.T) {
	mockAPI := newSimAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSimularBot(mockAPI)
	ctx := context.Background()

	// 1. Comando /simular sem parâmetros
	updateCmd := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/simular",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateCmd)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Simulador What-If de Compras")
	assert.Contains(t, lastMsg.Text, "Sintaxe: `/simular <valor> [parcelas]`")
	assert.Contains(t, lastMsg.Text, "• `/simular 2400 12`")

	// 2. Botão persistente "🔮 Simular Compra"
	updateBtn := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: bot.ButtonSimulate,
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateBtn)

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Simulador What-If de Compras")
}

func TestSimular_InvalidArguments(t *testing.T) {
	mockAPI := newSimAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSimularBot(mockAPI)
	ctx := context.Background()

	updateInvalid := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/simular texto_invalido",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateInvalid)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "⚠️")
	assert.Contains(t, lastMsg.Text, "Exemplo correto: `/simular 2400 12`")
}

func TestSimular_SuccessFlow_WithExecutiveSummary_AndConfirmation(t *testing.T) {
	mockAPI := newSimAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSimularBot(mockAPI)
	ctx := context.Background()

	// Simulação: /simular 2400 12
	updateSim := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/simular 2400 12",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateSim)

	mockAPI.mu.Lock()
	assert.NotNil(t, mockAPI.lastSimReq)
	assert.Equal(t, float64(2400), mockAPI.lastSimReq.Amount)
	assert.Equal(t, 12, mockAPI.lastSimReq.Installments)
	mockAPI.mu.Unlock()

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	// Resumo Executivo (D-05, D-06)
	assert.Contains(t, lastMsg.Text, "R$ 2400,00 (12x de R$ 200,00)")
	assert.Contains(t, lastMsg.Text, "S2S cai de R$ 100,00 para *R$ 80,00/dia* (-R$ 20,00/dia)")
	assert.Contains(t, lastMsg.Text, "Ciclo 4 (S2S projetado: R$ 45,00/dia — 🟡 ATENÇÃO)")

	// Verifica Teclado Inline com botão de confirmação e cancelamento (D-07)
	inlineKb, isInline := lastMsg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	require.True(t, isInline)
	require.Len(t, inlineKb.InlineKeyboard, 1)
	assert.Equal(t, "✅ Confirmar e Lançar", inlineKb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "sim_confirm:2400.00:12", *inlineKb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "❌ Cancelar", inlineKb.InlineKeyboard[0][1].Text)
	assert.Equal(t, "sim_cancel", *inlineKb.InlineKeyboard[0][1].CallbackData)

	// Ação de confirmação via callback (D-07)
	updateConfirm := &tgbotapi.Update{
		UpdateID: 2,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-confirm-1",
			Data: "sim_confirm:2400.00:12",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Message: &tgbotapi.Message{
				MessageID: 1,
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}
	testBot.ProcessUpdate(ctx, updateConfirm)

	mockAPI.mu.Lock()
	assert.NotNil(t, mockAPI.lastQuickReq)
	assert.Equal(t, float64(200), mockAPI.lastQuickReq.Amount)
	assert.Equal(t, "Compra simulada (1/12)", mockAPI.lastQuickReq.Description)
	assert.Equal(t, "Outros", mockAPI.lastQuickReq.CategoryName)
	mockAPI.mu.Unlock()

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Compra confirmada e lançada com sucesso!")
	assert.Contains(t, lastMsg.Text, "*R$ 2400,00* em 12x de *R$ 200,00*")
}

func TestSimular_DeficitRiskAlert(t *testing.T) {
	mockAPI := newSimAPIServerMock()
	defer mockAPI.server.Close()

	mockAPI.mu.Lock()
	mockAPI.simulateResp.DeficitRiskAlert = true
	mockAPI.simulateResp.CriticalCycleNumber = 5
	mockAPI.simulateResp.CriticalCycleS2S = -15.50
	mockAPI.simulateResp.CriticalCycleHealthStatus = "DEFICIT"
	mockAPI.simulateResp.RecommendationMessage = "Recomendamos estender para 18 parcelas."
	mockAPI.mu.Unlock()

	testBot, sender := setupSimularBot(mockAPI)
	ctx := context.Background()

	updateSim := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/simular 5000 6",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateSim)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	// Alerta Preventivo de Risco de Déficit (D-08)
	assert.Contains(t, lastMsg.Text, "🚨 *ALERTA DE RISCO DE DÉFICIT:*")
	assert.Contains(t, lastMsg.Text, "O Ciclo 5 fechará no vermelho com essa despesa!")
	assert.Contains(t, lastMsg.Text, "Recomendamos estender para 18 parcelas.")
	assert.Contains(t, lastMsg.Text, "🔴 DÉFICIT")
}

func TestSimular_CancelCallback(t *testing.T) {
	mockAPI := newSimAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSimularBot(mockAPI)
	ctx := context.Background()

	updateCancel := &tgbotapi.Update{
		UpdateID: 1,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-cancel-sim",
			Data: "sim_cancel",
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
	assert.Contains(t, lastMsg.Text, "Simulação cancelada. Nenhuma transação foi lançada.")
}
