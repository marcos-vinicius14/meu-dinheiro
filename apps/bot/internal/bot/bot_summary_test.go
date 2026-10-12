package bot_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type summaryAPIServerMock struct {
	server    *httptest.Server
	todayResp client.SummaryReportResponse
	weekResp  client.SummaryReportResponse
}

func newSummaryAPIServerMock() *summaryAPIServerMock {
	mock := &summaryAPIServerMock{
		todayResp: client.SummaryReportResponse{
			Period:             "today",
			AccountBalance:     4500.00,
			TotalLiquidBalance: 4500.00,
			S2SToday:           78.50,
			HealthStatus:       "HEALTHY",
			DaysRemaining:      22,
			Today: &client.DaySummaryResponse{
				Date:              "2026-10-12",
				TotalSpent:        50.50,
				TransactionsCount: 2,
				Transactions: []client.DaySummaryItem{
					{
						ID:           "tx-1",
						Description:  "Café e Pão de Queijo",
						Amount:       15.00,
						CategoryName: "Alimentação",
						Type:         "FLEXIBLE_EXPENSE",
						Date:         "2026-10-12",
					},
					{
						ID:           "tx-2",
						Description:  "Almoço Executivo",
						Amount:       35.50,
						CategoryName: "Alimentação",
						Type:         "FLEXIBLE_EXPENSE",
						Date:         "2026-10-12",
					},
				},
			},
		},
		weekResp: client.SummaryReportResponse{
			Period:             "week",
			AccountBalance:     4500.00,
			TotalLiquidBalance: 4500.00,
			S2SToday:           78.50,
			HealthStatus:       "HEALTHY",
			DaysRemaining:      22,
			Week: &client.WeekSummaryResponse{
				StartDate:         "2026-10-12",
				EndDate:           "2026-10-18",
				DaysElapsed:       3,
				TotalSpent:        400.00,
				DailyAverage:      133.33,
				TransactionsCount: 4,
				TopCategories: []client.CategorySummaryItem{
					{
						CategoryID:        "c-1",
						CategoryName:      "Alimentação",
						Amount:            250.00,
						Percentage:        62.5,
						TransactionsCount: 2,
					},
					{
						CategoryID:        "c-2",
						CategoryName:      "Transporte",
						Amount:            100.00,
						Percentage:        25.0,
						TransactionsCount: 1,
					},
					{
						CategoryID:        "c-3",
						CategoryName:      "Lazer",
						Amount:            50.00,
						Percentage:        12.5,
						TransactionsCount: 1,
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/internal/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		tgID := r.URL.Query().Get("telegram_id")
		period := r.URL.Query().Get("period")

		if tgID == "99999" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuário não encontrado"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if period == "week" {
			_ = json.NewEncoder(w).Encode(mock.weekResp)
		} else {
			_ = json.NewEncoder(w).Encode(mock.todayResp)
		}
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func setupSummaryBot(apiMock *summaryAPIServerMock) (*bot.Bot, *mockSender) {
	sender := &mockSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiMock.server.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, &config.Config{})
	b.SetSender(sender)
	return b, sender
}

func TestBot_SummaryDaily_Success(t *testing.T) {
	mockAPI := newSummaryAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSummaryBot(mockAPI)
	ctx := context.Background()

	// 1. Testa comando /resumo
	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/resumo",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "📅 *Resumo de Hoje (12/10/2026)*")
	assert.Contains(t, lastMsg.Text, "💳 *Saldo em Conta:* R$ 4500,00")
	assert.Contains(t, lastMsg.Text, "💸 *Total Gasto Hoje:* R$ 50,50 (2 lançamentos)")
	assert.Contains(t, lastMsg.Text, "💰 *S2S para Hoje:* R$ 78,50/dia")
	assert.Contains(t, lastMsg.Text, "📊 *Status do Ciclo:* 🟢 SAUDÁVEL (22 dias restantes)")
	assert.Contains(t, lastMsg.Text, "📋 *Extrato Rápido do Dia:*")
	assert.Contains(t, lastMsg.Text, "• Café e Pão de Queijo (Alimentação) — R$ 15,00")
	assert.Contains(t, lastMsg.Text, "• Almoço Executivo (Alimentação) — R$ 35,50")

	// 2. Testa comando /resumo hoje
	updateHoje := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/resumo hoje",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateHoje)

	lastMsgHoje, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsgHoje.Text, "📅 *Resumo de Hoje (12/10/2026)*")
	assert.Contains(t, lastMsgHoje.Text, "💸 *Total Gasto Hoje:* R$ 50,50")
}

func TestBot_SummaryWeekly_Success(t *testing.T) {
	mockAPI := newSummaryAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSummaryBot(mockAPI)
	ctx := context.Background()

	// 1. Testa comando /semanal
	updateSemanal := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/semanal",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateSemanal)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "📊 *Resumo Semanal (12/10 a 18/10)*")
	assert.Contains(t, lastMsg.Text, "💳 *Saldo Atual em Conta:* R$ 4500,00")
	assert.Contains(t, lastMsg.Text, "💸 *Total Gasto na Semana:* R$ 400,00 (4 lançamentos)")
	assert.Contains(t, lastMsg.Text, "📉 *Média Diária (3 dias decorridos):* R$ 133,33/dia")
	assert.Contains(t, lastMsg.Text, "💰 *S2S Diário Atual:* R$ 78,50/dia")
	assert.Contains(t, lastMsg.Text, "📊 *Status do Ciclo:* 🟢 SAUDÁVEL")
	assert.Contains(t, lastMsg.Text, "🏷️ *Top Categorias de Gastos:*")
	assert.Contains(t, lastMsg.Text, "1. Alimentação: R$ 250,00 (62,5%)")
	assert.Contains(t, lastMsg.Text, "2. Transporte: R$ 100,00 (25,0%)")
	assert.Contains(t, lastMsg.Text, "3. Lazer: R$ 50,00 (12,5%)")

	// 2. Testa comando /resumo semanal
	updateResumoSemanal := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/resumo semanal",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, updateResumoSemanal)

	lastMsg2, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg2.Text, "📊 *Resumo Semanal (12/10 a 18/10)*")
}

func TestBot_Summary_InvalidArgs(t *testing.T) {
	mockAPI := newSummaryAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSummaryBot(mockAPI)
	ctx := context.Background()

	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/resumo mes",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "período inválido")
	assert.Contains(t, lastMsg.Text, "/resumo hoje")
	assert.Contains(t, lastMsg.Text, "/semanal")
}

func TestBot_Summary_UserNotFound(t *testing.T) {
	mockAPI := newSummaryAPIServerMock()
	defer mockAPI.server.Close()

	testBot, sender := setupSummaryBot(mockAPI)
	ctx := context.Background()

	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/resumo",
			From: &tgbotapi.User{ID: 99999, FirstName: "Desconhecido"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	assert.Contains(t, lastMsg.Text, "Você ainda não possui cadastro no *Meu Dinheiro*")
}

func TestBot_Summary_MarkdownEscaping(t *testing.T) {
	mockAPI := newSummaryAPIServerMock()
	mockAPI.todayResp.Today.Transactions = []client.DaySummaryItem{
		{
			ID:           "tx-esc",
			Description:  "Almoço_executivo*premium [promo]",
			Amount:       45.00,
			CategoryName: "Alimentação_Geral*",
			Type:         "FLEXIBLE_EXPENSE",
			Date:         "2026-10-12",
		},
	}
	defer mockAPI.server.Close()

	testBot, sender := setupSummaryBot(mockAPI)
	ctx := context.Background()

	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/resumo",
			From: &tgbotapi.User{ID: 12345, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: 100},
		},
	}
	testBot.ProcessUpdate(ctx, update)

	lastMsg, ok := sender.LastMessage()
	require.True(t, ok)
	// Caracteres _, * e [ devem estar escapados com barra invertida
	assert.Contains(t, lastMsg.Text, "Almoço\\_executivo\\*premium \\[promo\\]")
	assert.Contains(t, lastMsg.Text, "Alimentação\\_Geral\\*")
}
