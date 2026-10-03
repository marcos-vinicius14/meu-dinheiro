package fsm

import (
	"context"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

// TelegramSender define os métodos necessários para envio de mensagens e callbacks ao Telegram,
// permitindo a injeção de fakes/mocks em testes unitários.
type TelegramSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// FSM é o motor da máquina de estados do onboarding no Telegram Bot.
type FSM struct {
	store   *SessionStore
	sender  TelegramSender
	client  *client.APIClient
	logger  *slog.Logger
	handler StepHandler
}

// StepHandler define a interface para tratamento das mensagens e callbacks de cada estado do diálogo.
type StepHandler interface {
	HandleStepMessage(ctx context.Context, f *FSM, sess *Session, text string) error
	HandleStepCallback(ctx context.Context, f *FSM, sess *Session, data string) error
	HandleStart(ctx context.Context, f *FSM, sess *Session) error
}

// NewFSM instancia o motor da máquina de estados.
func NewFSM(store *SessionStore, sender TelegramSender, client *client.APIClient, logger *slog.Logger) *FSM {
	return &FSM{
		store:  store,
		sender: sender,
		client: client,
		logger: logger,
	}
}

// SetStepHandler registra o tratador das etapas de onboarding.
func (f *FSM) SetStepHandler(h StepHandler) {
	f.handler = h
}

// GetStore expõe o SessionStore associado.
func (f *FSM) GetStore() *SessionStore {
	return f.store
}

// SetSender permite atualizar o emissor de mensagens (útil para testes).
func (f *FSM) SetSender(sender TelegramSender) {
	f.sender = sender
}

// Reply envia uma mensagem de texto simples formatada em Markdown para o chat especificado.
func (f *FSM) Reply(chatID int64, text string) {
	if f.sender == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := f.sender.Send(msg); err != nil {
		f.logger.Error("falha ao enviar mensagem telegram", "chat_id", chatID, "error", err)
		if strings.Contains(err.Error(), "can't parse entities") {
			msg.ParseMode = ""
			_, _ = f.sender.Send(msg)
		}
	}
}

// ReplyWithKeyboard envia uma mensagem com teclado inline para o chat especificado.
func (f *FSM) ReplyWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	if f.sender == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	if _, err := f.sender.Send(msg); err != nil {
		f.logger.Error("falha ao enviar teclado telegram", "chat_id", chatID, "error", err)
		if strings.Contains(err.Error(), "can't parse entities") {
			msg.ParseMode = ""
			_, _ = f.sender.Send(msg)
		}
	}
}

// HandleUpdate intercepta e processa mensagens ou callbacks direcionados à FSM de onboarding.
// Retorna handled = true caso a FSM tenha consumido a atualização.
func (f *FSM) HandleUpdate(ctx context.Context, update *tgbotapi.Update) (bool, error) {
	if update == nil {
		return false, nil
	}

	// 1. Tratamento de CallbackQuery (botões inline)
	if update.CallbackQuery != nil {
		cb := update.CallbackQuery
		if f.sender != nil {
			_, _ = f.sender.Request(tgbotapi.NewCallback(cb.ID, ""))
		}

		user := cb.From
		sess, exists := f.store.Get(user.ID)
		if !exists || sess.CurrentState == StateIdle {
			f.Reply(cb.Message.Chat.ID, "Sua sessão de onboarding expirou ou não está ativa. Digite /start para iniciar.")
			return true, nil
		}

		f.store.Touch(user.ID)

		if f.handler != nil {
			err := f.handler.HandleStepCallback(ctx, f, sess, cb.Data)
			return true, err
		}

		return true, nil
	}

	// 2. Tratamento de Mensagens de Texto
	if update.Message == nil {
		return false, nil
	}

	msg := update.Message
	text := strings.TrimSpace(msg.Text)
	user := msg.From
	if user == nil {
		return false, nil
	}

	if strings.HasPrefix(text, "/start auth_") {
		return false, nil
	}

	if text == "/cancelar" {
		f.store.Delete(user.ID)
		f.Reply(msg.Chat.ID, "❌ Onboarding cancelado. Você pode reiniciar a qualquer momento enviando /start.")
		return true, nil
	}

	sess, exists := f.store.Get(user.ID)

	if text == "/start" {
		if !exists {
			sess = f.store.GetOrCreate(user.ID, msg.Chat.ID, user.FirstName, user.UserName)
		} else {
			sess.ChatID = msg.Chat.ID
			sess.FirstName = user.FirstName
			sess.Username = user.UserName
			sess.CurrentState = StateIdle
			f.store.Touch(user.ID)
		}

		if f.handler != nil {
			err := f.handler.HandleStart(ctx, f, sess)
			return true, err
		}

		return true, nil
	}

	if exists && sess.CurrentState != StateIdle {
		f.store.Touch(user.ID)
		if f.handler != nil {
			err := f.handler.HandleStepMessage(ctx, f, sess, text)
			return true, err
		}
		return true, nil
	}

	return false, nil
}
