package bot_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSender struct {
	mu           sync.Mutex
	sentMessages []tgbotapi.Chattable
	requests     []tgbotapi.Chattable
}

func (m *mockSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, c)
	return tgbotapi.Message{MessageID: 1}, nil
}

func (m *mockSender) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, c)
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockSender) LastText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sentMessages) == 0 {
		return ""
	}
	last := m.sentMessages[len(m.sentMessages)-1]
	if msg, ok := last.(tgbotapi.MessageConfig); ok {
		return msg.Text
	}
	return ""
}

func TestBot_OnboardingIntegration_E2E(t *testing.T) {
	var mu sync.Mutex
	var lastReq *client.OnboardingRequest

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/users/context-by-telegram" {
			// Novo usuário retorna 404
			http.Error(w, `{"error":"não encontrado"}`, http.StatusNotFound)
			return
		}
		if r.URL.Path == "/internal/users/onboarding" {
			var req client.OnboardingRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			lastReq = &req
			mu.Unlock()

			resp := client.OnboardingResponse{
				Message: "Onboarding concluído com sucesso",
				UserID:  "u-integration-123",
				Cycle: client.CycleResponse{
					StartDate:     "2026-10-01",
					EndDate:       "2026-10-31",
					DaysRemaining: 28,
					S2SToday:      100.00,
					HealthStatus:  "HEALTHY",
				},
				EmergencyFund: client.EmergencyFundResponse{
					MonthlyEssentialCost: 1000.00,
					ChosenTarget:         12000.00,
					ChosenMonths:         12,
					MonthsCovered:        4.0,
					ProgressPercent:      33.3,
				},
				TotalLiquidBalance: 4000.00,
				TotalInvested:      0,
				TotalNetWorth:      4000.00,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer apiServer.Close()

	cfg := &config.Config{
		WebhookPath: "/webhook",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiServer.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, cfg)

	sender := &mockSender{}
	b.FSM().SetSender(sender)
	handler := b.NewWebhookHandler()

	sendUpdate := func(update tgbotapi.Update) {
		data, err := json.Marshal(update)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(data))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	telegramID := int64(7777)
	chatID := int64(8888)

	// 1. Envia /start
	sendUpdate(tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})

	sess, exists := b.SessionStore().Get(telegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateWaitingBalance, sess.CurrentState)
	assert.Contains(t, sender.LastText(), "saldo total")

	// 2. Envia saldo: 4000.00
	sendUpdate(tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "4000.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingCycleDay, sess.CurrentState)
	assert.Equal(t, 4000.00, sess.InitialBalance)

	// 3. Envia dia do ciclo: 1
	sendUpdate(tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "1",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingFixedExpenses, sess.CurrentState)
	assert.Equal(t, 1, sess.CycleStartDay)

	// 4. Envia despesa fixa: Mercado 1000
	sendUpdate(tgbotapi.Update{
		UpdateID: 4,
		Message: &tgbotapi.Message{
			Text: "Mercado 1000",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Len(t, sess.FixedExpenses, 1)
	assert.Equal(t, 1000.00, sess.MonthlyEssentialCost)

	// 5. Clica no callback: finish_fixed_expenses
	sendUpdate(tgbotapi.Update{
		UpdateID: 5,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_1",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Data: "finish_fixed_expenses",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingEmergencyFundChoice, sess.CurrentState)

	// 6. Clica em fund_12 (12 meses PJ)
	sendUpdate(tgbotapi.Update{
		UpdateID: 6,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_2",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Data: "fund_12",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingSavingsTarget, sess.CurrentState)
	assert.Equal(t, 12, sess.EmergencyFundMonths)

	// 7. Envia aporte mensal: 600.00
	sendUpdate(tgbotapi.Update{
		UpdateID: 7,
		Message: &tgbotapi.Message{
			Text: "600.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingInvestmentChoice, sess.CurrentState)
	assert.Equal(t, 600.00, sess.TargetSavings)

	// 8. Clica em skip_investments
	sendUpdate(tgbotapi.Update{
		UpdateID: 8,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_3",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Data: "skip_investments",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	})

	// Sessão deve estar limpa
	_, exists = b.SessionStore().Get(telegramID)
	assert.False(t, exists, "sessão deve ser removida após conclusão via webhook")

	// Payload enviado para a API
	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, lastReq)
	assert.Equal(t, telegramID, lastReq.TelegramID)
	assert.Equal(t, 4000.00, lastReq.InitialBalance)
	assert.Equal(t, 1, lastReq.CycleStartDay)
	assert.Equal(t, 12, lastReq.EmergencyFundMonths)
	assert.Equal(t, 600.00, lastReq.TargetSavings)
	assert.Len(t, lastReq.FixedExpenses, 1)

	// Mensagem de sucesso recebida
	assert.Contains(t, sender.LastText(), "Configuração Concluída com Sucesso!")
}
