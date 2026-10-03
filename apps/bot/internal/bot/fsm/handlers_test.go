package fsm_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiMockServer struct {
	mu           sync.Mutex
	server       *httptest.Server
	lastSaveReq  *client.OnboardingRequest
	saveCallFlag bool
}

func newAPIMockServer() *apiMockServer {
	m := &apiMockServer{}
	mux := http.NewServeMux()

	mux.HandleFunc("/internal/users/context-by-telegram", func(w http.ResponseWriter, r *http.Request) {
		tgID := r.URL.Query().Get("telegram_id")
		if tgID == "99999" {
			// Usuário já cadastrado
			resp := client.UserFinancialContext{}
			resp.User.ID = "u-999"
			resp.User.TelegramID = 99999
			resp.User.FirstName = "Marcos"
			resp.TotalLiquidBalance = 4500.00
			resp.Cycle.StartDate = "2026-10-05"
			resp.Cycle.EndDate = "2026-11-04"
			resp.Cycle.DaysRemaining = 20
			resp.Cycle.S2SToday = 85.50
			resp.Cycle.HealthStatus = "HEALTHY"
			resp.EmergencyFund.MonthsCovered = 2.5
			resp.EmergencyFund.ProgressPercent = 50.0
			resp.TotalNetWorth = 14500.00

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Usuário novo
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"usuário não encontrado"}`))
	})

	mux.HandleFunc("/internal/users/onboarding", func(w http.ResponseWriter, r *http.Request) {
		var req client.OnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		m.mu.Lock()
		m.lastSaveReq = &req
		m.saveCallFlag = true
		m.mu.Unlock()

		resp := client.OnboardingResponse{
			Message: "Onboarding concluído com sucesso",
			UserID:  "u-new-123",
			Cycle: client.CycleResponse{
				StartDate:     "2026-10-05",
				EndDate:       "2026-11-04",
				DaysRemaining: 30,
				S2SToday:      75.00,
				HealthStatus:  "HEALTHY",
			},
			EmergencyFund: client.EmergencyFundResponse{
				MonthlyEssentialCost: 1350.00,
				Suggested6x:          8100.00,
				Suggested12x:         16200.00,
				ChosenTarget:         8100.00,
				ChosenMonths:         req.EmergencyFundMonths,
				MonthsCovered:        2.59,
				ProgressPercent:      43.2,
			},
			TotalLiquidBalance: req.InitialBalance,
			TotalInvested:      422.30,
			TotalNetWorth:      req.InitialBalance + 422.30,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	m.server = httptest.NewServer(mux)
	return m
}

func setupFSMWithHandler(mockAPI *apiMockServer) (*fsm.FSM, *fsm.SessionStore, *mockTelegramSender) {
	store := fsm.NewSessionStore(10 * time.Minute)
	sender := &mockTelegramSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(mockAPI.server.URL, "internal-secret")
	engine := fsm.NewFSM(store, sender, apiClient, logger)
	handler := fsm.NewOnboardingHandler(apiClient, logger)
	engine.SetStepHandler(handler)
	return engine, store, sender
}

func getLastSentText(sender *mockTelegramSender) string {
	if len(sender.sentMessages) == 0 {
		return ""
	}
	last := sender.sentMessages[len(sender.sentMessages)-1]
	if msg, ok := last.(tgbotapi.MessageConfig); ok {
		return msg.Text
	}
	return ""
}

func TestOnboarding_HappyPath_WithoutInvestments(t *testing.T) {
	api := newAPIMockServer()
	defer api.server.Close()

	engine, store, sender := setupFSMWithHandler(api)
	ctx := context.Background()
	telegramID := int64(1001)
	chatID := int64(2001)

	// 1. /start
	updateStart := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice", UserName: "alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err := engine.HandleUpdate(ctx, updateStart)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, exists := store.Get(telegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateWaitingBalance, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "saldo total")

	// 2. Envia Saldo
	updateBalance := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "3500.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateBalance)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingCycleDay, sess.CurrentState)
	assert.Equal(t, 3500.00, sess.InitialBalance)
	assert.Contains(t, getLastSentText(sender), "ciclo mensal")

	// 3. Envia Dia do Ciclo
	updateCycle := &tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "5",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateCycle)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingFixedExpenses, sess.CurrentState)
	assert.Equal(t, 5, sess.CycleStartDay)
	assert.Contains(t, getLastSentText(sender), "despesas essenciais")

	// 4. Cadastra despesa 1: Aluguel 1200
	updateExp1 := &tgbotapi.Update{
		UpdateID: 4,
		Message: &tgbotapi.Message{
			Text: "Aluguel 1200",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateExp1)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Len(t, sess.FixedExpenses, 1)
	assert.Equal(t, 1200.00, sess.MonthlyEssentialCost)

	// 5. Cadastra despesa 2: Internet 150
	updateExp2 := &tgbotapi.Update{
		UpdateID: 5,
		Message: &tgbotapi.Message{
			Text: "Internet 150",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateExp2)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Len(t, sess.FixedExpenses, 2)
	assert.Equal(t, 1350.00, sess.MonthlyEssentialCost)

	// 6. Callback finish_fixed_expenses
	updateFinishFixed := &tgbotapi.Update{
		UpdateID: 6,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_1",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Data: "finish_fixed_expenses",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateFinishFixed)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingEmergencyFundChoice, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "Reserva de Emergência")

	// 7. Callback fund_6 (escolha de 6 meses)
	updateFund6 := &tgbotapi.Update{
		UpdateID: 7,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_2",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Data: "fund_6",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateFund6)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingSavingsTarget, sess.CurrentState)
	assert.Equal(t, 6, sess.EmergencyFundMonths)

	// 8. Envia Meta de Aporte
	updateSavings := &tgbotapi.Update{
		UpdateID: 8,
		Message: &tgbotapi.Message{
			Text: "300.00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateSavings)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingInvestmentChoice, sess.CurrentState)
	assert.Equal(t, 300.00, sess.TargetSavings)

	// 9. Callback skip_investments
	updateSkipInvest := &tgbotapi.Update{
		UpdateID: 9,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_3",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Alice"},
			Data: "skip_investments",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateSkipInvest)
	require.NoError(t, err)
	assert.True(t, handled)

	// Sessão deve ter sido expurgada do store após conclusão com sucesso
	_, exists = store.Get(telegramID)
	assert.False(t, exists, "sessão deve ser removida ao concluir onboarding")

	// Verifica se a API recebeu o payload correto
	api.mu.Lock()
	defer api.mu.Unlock()
	require.True(t, api.saveCallFlag)
	require.NotNil(t, api.lastSaveReq)
	assert.Equal(t, telegramID, api.lastSaveReq.TelegramID)
	assert.Equal(t, 3500.00, api.lastSaveReq.InitialBalance)
	assert.Equal(t, 5, api.lastSaveReq.CycleStartDay)
	assert.Len(t, api.lastSaveReq.FixedExpenses, 2)
	assert.Equal(t, 6, api.lastSaveReq.EmergencyFundMonths)
	assert.Equal(t, 300.00, api.lastSaveReq.TargetSavings)
	assert.Empty(t, api.lastSaveReq.Investments)

	// Verifica relatório final enviado
	lastMsg := getLastSentText(sender)
	assert.Contains(t, lastMsg, "Configuração Concluída com Sucesso!")
	assert.Contains(t, lastMsg, "S2S (Saldo Seguro Diário)")
	assert.Contains(t, lastMsg, "SAUDÁVEL")
}

func TestOnboarding_WithInvestments(t *testing.T) {
	api := newAPIMockServer()
	defer api.server.Close()

	engine, store, sender := setupFSMWithHandler(api)
	ctx := context.Background()
	telegramID := int64(1002)
	chatID := int64(2002)

	// Prepara sessão no estado de escolha de investimentos
	sess := store.GetOrCreate(telegramID, chatID, "Bob", "")
	sess.InitialBalance = 5000.00
	sess.CycleStartDay = 10
	sess.EmergencyFundMonths = 12
	sess.TargetSavings = 500.00
	sess.CurrentState = fsm.StateWaitingInvestmentChoice
	store.Set(sess)

	// 1. Clica em add_investment
	updateAddInv := &tgbotapi.Update{
		UpdateID: 10,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_inv_1",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Bob"},
			Data: "add_investment",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}
	handled, err := engine.HandleUpdate(ctx, updateAddInv)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingInvestmentInput, sess.CurrentState)

	// 2. Envia ticker e quantidade
	updateInvInput := &tgbotapi.Update{
		UpdateID: 11,
		Message: &tgbotapi.Message{
			Text: "ALUP11 10 42.23",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Bob"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateInvInput)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	require.Len(t, sess.Investments, 1)
	assert.Equal(t, "ALUP11", sess.Investments[0].Ticker)
	assert.Equal(t, 10.0, sess.Investments[0].Quantity)
	assert.Equal(t, 42.23, sess.Investments[0].AveragePrice)

	// 3. Clica em finish_investments
	updateFinishInv := &tgbotapi.Update{
		UpdateID: 12,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_inv_finish",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Bob"},
			Data: "finish_investments",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateFinishInv)
	require.NoError(t, err)
	assert.True(t, handled)

	// Sessão deve ser expurgada
	_, exists := store.Get(telegramID)
	assert.False(t, exists)

	// Verifica se a API recebeu o ativo
	api.mu.Lock()
	defer api.mu.Unlock()
	require.True(t, api.saveCallFlag)
	require.Len(t, api.lastSaveReq.Investments, 1)
	assert.Equal(t, "ALUP11", api.lastSaveReq.Investments[0].Ticker)
	assert.Equal(t, 10.0, api.lastSaveReq.Investments[0].Quantity)
	assert.Equal(t, 42.23, api.lastSaveReq.Investments[0].AveragePrice)
	assert.Contains(t, getLastSentText(sender), "Configuração Concluída com Sucesso!")
}

func TestOnboarding_ExistingUser_DailyDashboard(t *testing.T) {
	api := newAPIMockServer()
	defer api.server.Close()

	engine, store, sender := setupFSMWithHandler(api)
	ctx := context.Background()
	existingTelegramID := int64(99999)
	chatID := int64(90001)

	update := &tgbotapi.Update{
		UpdateID: 20,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: existingTelegramID, FirstName: "Marcos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}

	handled, err := engine.HandleUpdate(ctx, update)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, exists := store.Get(existingTelegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState, "sessão de usuário já cadastrado deve permanecer em IDLE")

	lastMsg := getLastSentText(sender)
	assert.Contains(t, lastMsg, "Bem-vindo de volta ao *Meu Dinheiro*")
	assert.Contains(t, lastMsg, "Painel do Dia")
	assert.Contains(t, lastMsg, "S2S de Hoje:")
	assert.Contains(t, lastMsg, "SAUDÁVEL")
	assert.Contains(t, lastMsg, "/s2s")
	assert.Contains(t, lastMsg, "/gasto")
}

func TestOnboarding_ResilienceAndErrors(t *testing.T) {
	api := newAPIMockServer()
	defer api.server.Close()

	engine, store, sender := setupFSMWithHandler(api)
	ctx := context.Background()
	telegramID := int64(1003)
	chatID := int64(2003)

	// 1. /start
	updateStart := &tgbotapi.Update{
		UpdateID: 30,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Carlos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err := engine.HandleUpdate(ctx, updateStart)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ := store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingBalance, sess.CurrentState)

	// 2. Envia saldo inválido ("dez")
	updateInvalidBalance := &tgbotapi.Update{
		UpdateID: 31,
		Message: &tgbotapi.Message{
			Text: "dez reais",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Carlos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateInvalidBalance)
	require.NoError(t, err)
	assert.True(t, handled)

	// Permanece em StateWaitingBalance e envia aviso
	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingBalance, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "Valor monetário inválido")

	// 3. Envia saldo correto
	updateValidBalance := &tgbotapi.Update{
		UpdateID: 32,
		Message: &tgbotapi.Message{
			Text: "2500,00",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Carlos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateValidBalance)
	require.NoError(t, err)
	assert.True(t, handled)

	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingCycleDay, sess.CurrentState)

	// 4. Envia dia inválido (30 acima do teto de 28)
	updateInvalidDay := &tgbotapi.Update{
		UpdateID: 33,
		Message: &tgbotapi.Message{
			Text: "30",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Carlos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateInvalidDay)
	require.NoError(t, err)
	assert.True(t, handled)

	// Permanece em StateWaitingCycleDay e explica a regra de fevereiro
	sess, _ = store.Get(telegramID)
	assert.Equal(t, fsm.StateWaitingCycleDay, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "o teto é 28 para consistência de calendário com o mês de fevereiro")

	// 5. Envia /cancelar
	updateCancel := &tgbotapi.Update{
		UpdateID: 34,
		Message: &tgbotapi.Message{
			Text: "/cancelar",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Carlos"},
			Chat: &tgbotapi.Chat{ID: chatID},
		},
	}
	handled, err = engine.HandleUpdate(ctx, updateCancel)
	require.NoError(t, err)
	assert.True(t, handled)

	_, exists := store.Get(telegramID)
	assert.False(t, exists, "sessão deve ser deletada imediatamente ao receber /cancelar")
	assert.Contains(t, getLastSentText(sender), "Onboarding cancelado")
}

func TestOnboarding_CallbackStateGuard_ReplayProtection(t *testing.T) {
	api := newAPIMockServer()
	defer api.server.Close()

	engine, store, sender := setupFSMWithHandler(api)
	ctx := context.Background()
	telegramID := int64(1004)
	chatID := int64(2004)

	// Prepara sessão em StateWaitingInvestmentChoice
	sess := store.GetOrCreate(telegramID, chatID, "Diana", "")
	sess.InitialBalance = 4000.00
	sess.CycleStartDay = 1
	sess.EmergencyFundMonths = 6
	sess.TargetSavings = 250.00
	sess.CurrentState = fsm.StateWaitingInvestmentChoice
	store.Set(sess)

	// Usuário clica em botão de mensagem antiga ("fund_12" ou "finish_fixed_expenses")
	updateOldCallback := &tgbotapi.Update{
		UpdateID: 40,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_old_replay",
			From: &tgbotapi.User{ID: telegramID, FirstName: "Diana"},
			Data: string(fsm.ActionFund12),
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
			},
		},
	}

	handled, err := engine.HandleUpdate(ctx, updateOldCallback)
	require.NoError(t, err)
	assert.True(t, handled)

	// Estado DEVE permanecer inalterado em StateWaitingInvestmentChoice
	sess, exists := store.Get(telegramID)
	require.True(t, exists)
	assert.Equal(t, fsm.StateWaitingInvestmentChoice, sess.CurrentState, "estado não deve mudar quando callback for incompatível")
	assert.Equal(t, 6, sess.EmergencyFundMonths, "dados anteriores não devem ser corrompidos")

	// Mensagem de aviso deve ser enviada
	lastMsg := getLastSentText(sender)
	assert.Contains(t, lastMsg, "Essa opção pertence a uma etapa anterior ou não é mais válida")
}
