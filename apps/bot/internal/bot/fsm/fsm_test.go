package fsm_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTelegramSender struct {
	sentMessages []tgbotapi.Chattable
	requests     []tgbotapi.Chattable
}

func (m *mockTelegramSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.sentMessages = append(m.sentMessages, c)
	return tgbotapi.Message{MessageID: 1}, nil
}

func (m *mockTelegramSender) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.requests = append(m.requests, c)
	return &tgbotapi.APIResponse{Ok: true}, nil
}

type mockStepHandler struct {
	startCalls    int
	messageCalls  int
	callbackCalls int
	lastText      string
	lastData      string
}

func (h *mockStepHandler) HandleStart(ctx context.Context, f *fsm.FSM, sess *fsm.Session) error {
	h.startCalls++
	sess.CurrentState = fsm.StateWaitingBalance
	f.Reply(sess.ChatID, "Qual o seu saldo?")
	return nil
}

func (h *mockStepHandler) HandleStepMessage(ctx context.Context, f *fsm.FSM, sess *fsm.Session, text string) error {
	h.messageCalls++
	h.lastText = text
	return nil
}

func (h *mockStepHandler) HandleStepCallback(ctx context.Context, f *fsm.FSM, sess *fsm.Session, data string) error {
	h.callbackCalls++
	h.lastData = data
	return nil
}

func setupTestFSM() (*fsm.FSM, *fsm.SessionStore, *mockTelegramSender, *mockStepHandler) {
	store := fsm.NewSessionStore(10 * time.Minute)
	sender := &mockTelegramSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient("http://localhost:8080", "secret")
	engine := fsm.NewFSM(store, sender, apiClient, logger)
	handler := &mockStepHandler{}
	engine.SetStepHandler(handler)
	return engine, store, sender, handler
}

func TestFSM_HandleAuthBypass(t *testing.T) {
	engine, _, _, _ := setupTestFSM()

	update := &tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			Text: "/start auth_12345678",
			From: &tgbotapi.User{ID: 100, FirstName: "Alice"},
			Chat: &tgbotapi.Chat{ID: 200},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.False(t, handled, "login web /start auth_ deve ser ignorado pela FSM para processamento no bot")
}

func TestFSM_HandleStart(t *testing.T) {
	engine, store, sender, handler := setupTestFSM()

	update := &tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			Text: "/start",
			From: &tgbotapi.User{ID: 101, FirstName: "Bob", UserName: "bob123"},
			Chat: &tgbotapi.Chat{ID: 201},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, handler.startCalls)

	sess, exists := store.Get(101)
	require.True(t, exists)
	assert.Equal(t, fsm.StateWaitingBalance, sess.CurrentState)
	assert.NotEmpty(t, sender.sentMessages)
}

func TestFSM_HandleCancelar(t *testing.T) {
	engine, store, sender, _ := setupTestFSM()

	// Inicia sessão no meio do onboarding
	sess := store.GetOrCreate(102, 202, "Charlie", "")
	sess.CurrentState = fsm.StateWaitingCycleDay
	store.Set(sess)

	update := &tgbotapi.Update{
		UpdateID: 3,
		Message: &tgbotapi.Message{
			Text: "/cancelar",
			From: &tgbotapi.User{ID: 102, FirstName: "Charlie"},
			Chat: &tgbotapi.Chat{ID: 202},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.True(t, handled)

	// Sessão deve ter sido expurgada
	_, exists := store.Get(102)
	assert.False(t, exists)
	assert.NotEmpty(t, sender.sentMessages)
}

func TestFSM_HandleActiveStepMessage(t *testing.T) {
	engine, store, _, handler := setupTestFSM()

	sess := store.GetOrCreate(103, 203, "Dave", "")
	sess.CurrentState = fsm.StateWaitingBalance
	store.Set(sess)

	update := &tgbotapi.Update{
		UpdateID: 4,
		Message: &tgbotapi.Message{
			Text: "3500.50",
			From: &tgbotapi.User{ID: 103, FirstName: "Dave"},
			Chat: &tgbotapi.Chat{ID: 203},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, handler.messageCalls)
	assert.Equal(t, "3500.50", handler.lastText)
}

func TestFSM_HandleCallbackQuery(t *testing.T) {
	engine, store, sender, handler := setupTestFSM()

	sess := store.GetOrCreate(104, 204, "Eve", "")
	sess.CurrentState = fsm.StateWaitingEmergencyFundChoice
	store.Set(sess)

	update := &tgbotapi.Update{
		UpdateID: 5,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb_123",
			From: &tgbotapi.User{ID: 104, FirstName: "Eve"},
			Data: "fund_6",
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: 204},
			},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, handler.callbackCalls)
	assert.Equal(t, "fund_6", handler.lastData)
	assert.NotEmpty(t, sender.requests, "deve ter enviado ack de callback ao telegram")
}

func TestFSM_PassesThroughUnknownCommandsWhenIdle(t *testing.T) {
	engine, _, _, _ := setupTestFSM()

	update := &tgbotapi.Update{
		UpdateID: 6,
		Message: &tgbotapi.Message{
			Text: "/ajuda",
			From: &tgbotapi.User{ID: 105, FirstName: "Frank"},
			Chat: &tgbotapi.Chat{ID: 205},
		},
	}

	handled, err := engine.HandleUpdate(context.Background(), update)
	require.NoError(t, err)
	assert.False(t, handled, "comandos como /ajuda devem ser repassados se não houver onboarding ativo")
}
