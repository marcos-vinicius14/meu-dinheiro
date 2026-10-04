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

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkinAPIMockServer struct {
	mu            sync.Mutex
	server        *httptest.Server
	lastCheckin   *client.DailyCheckInRequest
	lastAdjust    *client.AdjustBalanceRequest
	checkinCount  int
	adjustedValue float64
}

func newCheckinAPIMockServer() *checkinAPIMockServer {
	m := &checkinAPIMockServer{
		adjustedValue: 4250.00,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/internal/users/context-by-telegram", func(w http.ResponseWriter, r *http.Request) {
		tgID := r.URL.Query().Get("telegram_id")
		if tgID == "12345" {
			resp := client.UserFinancialContext{}
			resp.User.ID = "u-12345"
			resp.User.TelegramID = 12345
			resp.User.FirstName = "Marcos"
			m.mu.Lock()
			resp.TotalLiquidBalance = m.adjustedValue
			m.mu.Unlock()
			resp.Cycle.StartDate = "2026-10-05"
			resp.Cycle.EndDate = "2026-11-04"
			resp.Cycle.DaysRemaining = 22
			resp.Cycle.S2SToday = 80.00
			resp.Cycle.HealthStatus = "HEALTHY"

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"não encontrado"}`))
	})

	mux.HandleFunc("/internal/bank-accounts/balance", func(w http.ResponseWriter, r *http.Request) {
		var req client.AdjustBalanceRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.mu.Lock()
		m.lastAdjust = &req
		m.adjustedValue = req.NewBalance
		m.mu.Unlock()

		resp := client.AdjustBalanceResponse{
			AccountID:          "acc-1",
			AccountName:        "Conta Corrente",
			PreviousBalance:    4250.00,
			NewBalance:         req.NewBalance,
			TotalLiquidBalance: req.NewBalance,
			Message:            "Saldo atualizado com sucesso",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/internal/transactions/checkin", func(w http.ResponseWriter, r *http.Request) {
		var req client.DailyCheckInRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.mu.Lock()
		m.lastCheckin = &req
		m.checkinCount++
		m.mu.Unlock()

		spentToday := 0.0
		for _, e := range req.UntrackedExpenses {
			spentToday += e.Amount
		}

		resp := client.DailyCheckInResponse{
			S2SCalculated: 80.00,
			SpentToday:    spentToday,
			DailyQuota:    80.00,
			DeltaSavings:  80.00 - spentToday,
			HealthStatus:  "HEALTHY",
			NextDayS2S:    82.50,
			DaysRemaining: 21,
			Message:       "Check-in realizado com sucesso",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	m.server = httptest.NewServer(mux)
	return m
}

func setupCheckinTestFSM(mockAPI *checkinAPIMockServer) (*fsm.FSM, *fsm.SessionStore, *mockTelegramSender, *fsm.CheckinHandler) {
	store := fsm.NewSessionStore(10 * time.Minute)
	sender := &mockTelegramSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient(mockAPI.server.URL, "internal-secret")
	engine := fsm.NewFSM(store, sender, apiClient, logger)
	handler := fsm.NewCheckinHandler(apiClient, logger)
	return engine, store, sender, handler
}

func TestCheckin_HappyPath_WithoutExpensesAndBalanceConfirmed(t *testing.T) {
	api := newCheckinAPIMockServer()
	defer api.server.Close()

	engine, store, sender, handler := setupCheckinTestFSM(api)
	ctx := context.Background()
	telegramID := int64(12345)
	chatID := int64(1001)

	sess := store.GetOrCreate(telegramID, chatID, "Marcos", "marcos")

	// 1. Inicia Check-in
	err := handler.StartCheckin(ctx, engine, sess)
	require.NoError(t, err)
	assert.Equal(t, fsm.StateCheckinWaitingExpenses, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "Conferência de Gastos")

	// 2. Clica em "Não, tudo certo" (ActionCheckinNoExpenses)
	err = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinNoExpenses))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateCheckinWaitingBalanceConfirm, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "Conferência de Saldo Bancário")
	assert.Contains(t, getLastSentText(sender), "4250,00")

	// 3. Clica em "Sim, confere" (ActionCheckinBalanceOK)
	err = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinBalanceOK))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState)

	lastMsg := getLastSentText(sender)
	assert.Contains(t, lastMsg, "Check-in Diário Concluído!")
	assert.Contains(t, lastMsg, "Você economizou R$ 80,00 hoje!")
	assert.Contains(t, lastMsg, "S2S Projetado para Amanhã:")
	assert.Contains(t, lastMsg, "82,50/dia")

	api.mu.Lock()
	assert.Equal(t, 1, api.checkinCount)
	assert.Empty(t, api.lastCheckin.UntrackedExpenses)
	api.mu.Unlock()
}

func TestCheckin_WithUntrackedExpense(t *testing.T) {
	api := newCheckinAPIMockServer()
	defer api.server.Close()

	engine, store, sender, handler := setupCheckinTestFSM(api)
	ctx := context.Background()
	telegramID := int64(12345)
	chatID := int64(1001)

	sess := store.GetOrCreate(telegramID, chatID, "Marcos", "marcos")

	// Inicia Check-in
	_ = handler.StartCheckin(ctx, engine, sess)

	// Clica em "Sim, lançar"
	err := handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinAddExpense))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateCheckinEnteringExpense, sess.CurrentState)

	// Envia despesa
	err = handler.HandleStepMessage(ctx, engine, sess, "45.00 Farmácia")
	require.NoError(t, err)
	assert.Len(t, sess.CheckinExpenses, 1)
	assert.Contains(t, getLastSentText(sender), "Item adicionado")
	assert.Contains(t, getLastSentText(sender), "Farmácia")

	// Clica em "Concluir Despesas"
	err = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinFinishExpenses))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateCheckinWaitingBalanceConfirm, sess.CurrentState)

	// Confirma saldo
	err = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinBalanceOK))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState)

	api.mu.Lock()
	assert.Equal(t, 1, api.checkinCount)
	require.Len(t, api.lastCheckin.UntrackedExpenses, 1)
	assert.Equal(t, 45.00, api.lastCheckin.UntrackedExpenses[0].Amount)
	assert.Equal(t, "Farmácia", api.lastCheckin.UntrackedExpenses[0].Description)
	api.mu.Unlock()
}

func TestCheckin_WithBalanceAdjustment(t *testing.T) {
	api := newCheckinAPIMockServer()
	defer api.server.Close()

	engine, store, sender, handler := setupCheckinTestFSM(api)
	ctx := context.Background()
	telegramID := int64(12345)
	chatID := int64(1001)

	sess := store.GetOrCreate(telegramID, chatID, "Marcos", "marcos")

	// Inicia Check-in e pula despesas
	_ = handler.StartCheckin(ctx, engine, sess)
	_ = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinNoExpenses))
	assert.Equal(t, fsm.StateCheckinWaitingBalanceConfirm, sess.CurrentState)

	// Clica em "Ajustar saldo"
	err := handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinBalanceAdjust))
	require.NoError(t, err)
	assert.Equal(t, fsm.StateCheckinEnteringBalance, sess.CurrentState)
	assert.Contains(t, getLastSentText(sender), "Informe o saldo real atual")

	// Envia novo saldo
	err = handler.HandleStepMessage(ctx, engine, sess, "4100.00")
	require.NoError(t, err)
	assert.Equal(t, fsm.StateIdle, sess.CurrentState)

	api.mu.Lock()
	assert.Equal(t, 1, api.checkinCount)
	require.NotNil(t, api.lastAdjust)
	assert.Equal(t, 4100.00, api.lastAdjust.NewBalance)
	api.mu.Unlock()

	lastMsg := getLastSentText(sender)
	assert.Contains(t, lastMsg, "Check-in Diário Concluído!")
}

func TestCheckin_Idempotency_SameDayMultipleRuns(t *testing.T) {
	api := newCheckinAPIMockServer()
	defer api.server.Close()

	engine, store, _, handler := setupCheckinTestFSM(api)
	ctx := context.Background()
	telegramID := int64(12345)
	chatID := int64(1001)

	sess := store.GetOrCreate(telegramID, chatID, "Marcos", "marcos")

	// Check-in 1
	_ = handler.StartCheckin(ctx, engine, sess)
	_ = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinNoExpenses))
	_ = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinBalanceOK))

	// Check-in 2 no mesmo dia
	_ = handler.StartCheckin(ctx, engine, sess)
	_ = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinNoExpenses))
	_ = handler.HandleStepCallback(ctx, engine, sess, string(fsm.ActionCheckinBalanceOK))

	api.mu.Lock()
	assert.Equal(t, 2, api.checkinCount)
	api.mu.Unlock()
}
