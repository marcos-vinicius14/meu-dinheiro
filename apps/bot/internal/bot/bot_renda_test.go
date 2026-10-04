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

type rendaAPIServerMock struct {
	server        *httptest.Server
	lastIncomeReq *client.IncomeRequest
	incomeResp    client.IncomeResponse
	mu            sync.Mutex
}

func newRendaAPIServerMock() *rendaAPIServerMock {
	mock := &rendaAPIServerMock{
		incomeResp: client.IncomeResponse{
			CategoryName:       "Renda",
			PreviousS2S:        50.00,
			NewS2S:             216.66,
			HealthStatus:       "HEALTHY",
			DaysRemaining:      20,
			TotalLiquidBalance: 5200.00,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/transactions/income", func(w http.ResponseWriter, r *http.Request) {
		var req client.IncomeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mock.mu.Lock()
		mock.lastIncomeReq = &req
		resp := mock.incomeResp
		mock.mu.Unlock()

		if req.TelegramID == 99999 {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário não encontrado"})
			return
		}

		resp.Transaction.Amount = req.Amount
		resp.Transaction.Description = req.Description

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func setupRendaBot(apiMock *rendaAPIServerMock) (*bot.Bot, *mockSender) {
	sender := &mockSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiMock.server.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, &config.Config{})
	b.SetSender(sender)
	return b, sender
}

func TestIncome_HelpGuide(t *testing.T) {
	mockAPI := newRendaAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupRendaBot(mockAPI)
	ctx := context.Background()

	// 1. /renda sem argumentos
	updateCmd := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/renda",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateCmd)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Registro de Entrada de Dinheiro")
	assert.Contains(t, lastMsg.Text, "Sintaxe: `/renda <valor> <descrição>`")
	assert.Contains(t, lastMsg.Text, "• `/renda 5000 Salário mensal`")

	// 2. /receita sem argumentos
	updateAlias := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/receita",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateAlias)

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Registro de Entrada de Dinheiro")
}

func TestIncome_InvalidArguments(t *testing.T) {
	mockAPI := newRendaAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupRendaBot(mockAPI)
	ctx := context.Background()

	updateInvalid := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/renda texto_sem_valor",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateInvalid)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "⚠️")
	assert.Contains(t, lastMsg.Text, "Exemplo correto: `/renda 5000 Salário mensal`")
}

func TestIncome_SuccessFlow_RendaAndReceita(t *testing.T) {
	mockAPI := newRendaAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupRendaBot(mockAPI)
	ctx := context.Background()

	// 1. /renda 5000 Salário mensal
	updateRenda := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/renda 5000 Salário mensal",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateRenda)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastIncomeReq)
	assert.Equal(t, float64(5000), mockAPI.lastIncomeReq.Amount)
	assert.Equal(t, "Salário mensal", mockAPI.lastIncomeReq.Description)
	mockAPI.mu.Unlock()

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Entrada de R$ 5000,00 registrada com sucesso!")
	assert.Contains(t, lastMsg.Text, "Descrição: *Salário mensal*")
	assert.Contains(t, lastMsg.Text, "S2S Anterior:* R$ 50,00/dia")
	assert.Contains(t, lastMsg.Text, "Novo S2S:* R$ 216,66/dia (*+R$ 166,66/dia*)")
	assert.Contains(t, lastMsg.Text, "Saldo Líquido Disponível:* R$ 5200,00")
	assert.Contains(t, lastMsg.Text, "🟢 SAUDÁVEL")

	// 2. /receita 450 Pix freela (alias)
	updateReceita := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/receita 450 Pix freela",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateReceita)

	mockAPI.mu.Lock()
	require.NotNil(t, mockAPI.lastIncomeReq)
	assert.Equal(t, float64(450), mockAPI.lastIncomeReq.Amount)
	assert.Equal(t, "Pix freela", mockAPI.lastIncomeReq.Description)
	mockAPI.mu.Unlock()

	lastMsg, ok = sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Entrada de R$ 450,00 registrada com sucesso!")
}
