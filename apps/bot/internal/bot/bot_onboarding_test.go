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

func (m *mockSender) LastMessage() (tgbotapi.MessageConfig, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sentMessages) == 0 {
		return tgbotapi.MessageConfig{}, false
	}
	last := m.sentMessages[len(m.sentMessages)-1]
	if msg, ok := last.(tgbotapi.MessageConfig); ok {
		return msg, true
	}
	return tgbotapi.MessageConfig{}, false
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
	assert.Equal(t, fsm.StateWaitingCurrentEmergencyFundChoice, sess.CurrentState)

	// 5b. Responde se já possui valor guardado: não possui
	sendUpdate(tgbotapi.Update{
		UpdateID: 6,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_1b",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Data: string(fsm.ActionNoEmergencyFund),
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	})
	sess, _ = b.SessionStore().Get(telegramID)
	assert.Equal(t, fsm.StateWaitingEmergencyFundChoice, sess.CurrentState)
	assert.Equal(t, 0.0, sess.CurrentEmergencyFund)

	// 6. Clica em fund_12 (12 meses PJ)
	sendUpdate(tgbotapi.Update{
		UpdateID: 7,
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
		UpdateID: 8,
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
		UpdateID: 9,
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
	assert.Equal(t, 0.0, lastReq.CurrentEmergencyFund)
	assert.Len(t, lastReq.FixedExpenses, 1)

	// Mensagem de sucesso recebida
	assert.Contains(t, sender.LastText(), "Configuração Concluída com Sucesso!")
}

func TestBot_OnboardingIntegration_WithInvestments(t *testing.T) {
	var mu sync.Mutex
	var lastReq *client.OnboardingRequest

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/users/context-by-telegram" {
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
				UserID:  "u-inv-123",
				Cycle: client.CycleResponse{
					StartDate:     "2026-10-10",
					EndDate:       "2026-11-09",
					DaysRemaining: 30,
					S2SToday:      120.00,
					HealthStatus:  "HEALTHY",
				},
				EmergencyFund: client.EmergencyFundResponse{
					MonthlyEssentialCost: 1800.00,
					ChosenTarget:         21600.00,
					ChosenMonths:         12,
					MonthsCovered:        3.33,
					ProgressPercent:      27.8,
				},
				TotalLiquidBalance: 6000.00,
				TotalInvested:      422.30,
				TotalNetWorth:      6422.30,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer apiServer.Close()

	cfg := &config.Config{WebhookPath: "/webhook"}
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

	telegramID := int64(9901)
	chatID := int64(9902)

	// 1. /start
	sendUpdate(tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 2. Saldo
	sendUpdate(tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "6000.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 3. Ciclo
	sendUpdate(tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "10",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 4. Despesa essencial
	sendUpdate(tgbotapi.Update{
		UpdateID: 4,
		Message: &tgbotapi.Message{
			Text: "Aluguel 1800",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 5. Conclui despesas
	sendUpdate(tgbotapi.Update{
		UpdateID: 5,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_fix",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Data:    string(fsm.ActionFinishFixedExpenses),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	})
	// 5b. Responde que já possui reserva guardada
	sendUpdate(tgbotapi.Update{
		UpdateID: 6,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_have_fund",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Data:    string(fsm.ActionHaveEmergencyFund),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	})
	// 5c. Envia valor já guardado: 2000.00
	sendUpdate(tgbotapi.Update{
		UpdateID: 7,
		Message: &tgbotapi.Message{
			Text: "2000.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 6. Escolhe reserva 12 meses
	sendUpdate(tgbotapi.Update{
		UpdateID: 8,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_f12",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Data:    string(fsm.ActionFund12),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	})
	// 7. Aporte mensal
	sendUpdate(tgbotapi.Update{
		UpdateID: 9,
		Message: &tgbotapi.Message{
			Text: "500.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 8. Escolhe adicionar investimento
	sendUpdate(tgbotapi.Update{
		UpdateID: 10,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_add_inv",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Data:    string(fsm.ActionAddInvestment),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	})
	// 9. Envia ativo
	sendUpdate(tgbotapi.Update{
		UpdateID: 11,
		Message: &tgbotapi.Message{
			Text: "ALUP11 10 42.23",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	})
	// 10. Conclui onboarding com investimentos
	sendUpdate(tgbotapi.Update{
		UpdateID: 12,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_finish_inv",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Arthur"},
			Data:    string(fsm.ActionFinishInvestments),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	})

	// Sessão deve estar limpa da memória
	_, exists := b.SessionStore().Get(telegramID)
	assert.False(t, exists)

	// Valida payload de investimentos na API
	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, lastReq)
	assert.Equal(t, telegramID, lastReq.TelegramID)
	assert.Equal(t, 6000.00, lastReq.InitialBalance)
	assert.Equal(t, 10, lastReq.CycleStartDay)
	assert.Equal(t, 12, lastReq.EmergencyFundMonths)
	assert.Equal(t, 500.00, lastReq.TargetSavings)
	assert.Equal(t, 2000.00, lastReq.CurrentEmergencyFund)
	require.Len(t, lastReq.Investments, 1)
	assert.Equal(t, "ALUP11", lastReq.Investments[0].Ticker)
	assert.Equal(t, 10.0, lastReq.Investments[0].Quantity)
	assert.Equal(t, 42.23, lastReq.Investments[0].AveragePrice)

	assert.Contains(t, sender.LastText(), "Configuração Concluída com Sucesso!")
}

func TestBot_OnboardingIntegration_ExistingUserDashboard(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/users/context-by-telegram" {
			resp := client.UserFinancialContext{}
			resp.User.ID = "u-existing-888"
			resp.User.TelegramID = 88888
			resp.User.FirstName = "Fernanda"
			resp.TotalLiquidBalance = 7500.00
			resp.Cycle.StartDate = "2026-10-01"
			resp.Cycle.EndDate = "2026-10-31"
			resp.Cycle.DaysRemaining = 15
			resp.Cycle.S2SToday = 95.00
			resp.Cycle.HealthStatus = "HEALTHY"
			resp.EmergencyFund.MonthsCovered = 4.2
			resp.EmergencyFund.ProgressPercent = 70.0
			resp.TotalNetWorth = 25000.00

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer apiServer.Close()

	cfg := &config.Config{WebhookPath: "/webhook"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiServer.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, cfg)

	sender := &mockSender{}
	b.FSM().SetSender(sender)
	handler := b.NewWebhookHandler()

	telegramID := int64(88888)
	chatID := int64(88889)

	update := tgbotapi.Update{
		UpdateID: 20,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Fernanda"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}

	data, err := json.Marshal(update)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	sess, exists := b.SessionStore().Get(telegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState)

	lastText := sender.LastText()
	assert.Contains(t, lastText, "Bem-vindo de volta ao *Meu Dinheiro*")
	assert.Contains(t, lastText, "S2S de Hoje:")
	assert.Contains(t, lastText, "R$ 95.00/dia")
	assert.Contains(t, lastText, "SAUDÁVEL")
}

func TestBot_OnboardingIntegration_APIErrorResilience(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/users/context-by-telegram" {
			http.Error(w, `{"error":"não encontrado"}`, http.StatusNotFound)
			return
		}
		if r.URL.Path == "/internal/users/onboarding" {
			http.Error(w, `{"error":"banco indisponível"}`, http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer apiServer.Close()

	cfg := &config.Config{WebhookPath: "/webhook"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(apiServer.URL, "test-secret")
	b := bot.NewBot(nil, apiClient, logger, cfg)

	sender := &mockSender{}
	b.FSM().SetSender(sender)
	handler := b.NewWebhookHandler()

	telegramID := int64(7788)
	chatID := int64(7789)

	sess := b.SessionStore().GetOrCreate(telegramID, chatID, "Renato", "")
	sess.InitialBalance = 3000.00
	sess.CycleStartDay = 1
	sess.EmergencyFundMonths = 6
	sess.TargetSavings = 300.00
	sess.CurrentState = fsm.StateWaitingInvestmentChoice
	b.SessionStore().Set(sess)

	update := tgbotapi.Update{
		UpdateID: 30,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_err",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Renato"},
			Data:    string(fsm.ActionSkipInvestments),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	}
	data, err := json.Marshal(update)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Sessão NÃO deve ser expurgada para permitir nova tentativa
	sessAfter, exists := b.SessionStore().Get(telegramID)
	require.True(t, exists, "sessão deve permanecer em caso de falha de rede da API")
	assert.Equal(t, 3000.00, sessAfter.InitialBalance)

	// Notificação de erro enviada ao usuário
	assert.Contains(t, sender.LastText(), "Ocorreu um erro ao salvar suas configurações")
}

func TestBot_OnboardingIntegration_CallbackReplayProtection(t *testing.T) {
	cfg := &config.Config{WebhookPath: "/webhook"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient("http://localhost:9999", "test-secret")
	b := bot.NewBot(nil, apiClient, logger, cfg)

	sender := &mockSender{}
	b.FSM().SetSender(sender)
	handler := b.NewWebhookHandler()

	telegramID := int64(5555)
	chatID := int64(5556)

	sess := b.SessionStore().GetOrCreate(telegramID, chatID, "Lucia", "")
	sess.InitialBalance = 5000.00
	sess.CurrentState = fsm.StateWaitingInvestmentInput
	b.SessionStore().Set(sess)

	// Recebe callback antigo de escolha de reserva (fund_6)
	update := tgbotapi.Update{
		UpdateID: 40,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:      "cb_replay",
			From:    &tgbotapi.User{ID: telegramID, FirstName: "Lucia"},
			Data:    string(fsm.ActionFund6),
			Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}},
		},
	}
	data, err := json.Marshal(update)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Estado deve continuar sendo StateWaitingInvestmentInput
	sessAfter, exists := b.SessionStore().Get(telegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateWaitingInvestmentInput, sessAfter.CurrentState)

	// Mensagem de aviso deve ser disparada
	assert.Contains(t, sender.LastText(), "Essa opção pertence a uma etapa anterior ou não é mais válida")
}
