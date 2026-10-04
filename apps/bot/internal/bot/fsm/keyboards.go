package fsm

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ButtonS2SToday     = "💰 S2S Hoje"
	ButtonQuickExpense = "💸 Lançar Gasto"
	ButtonSimulate     = "🔮 Simular"
	ButtonCheckin      = "📝 Check-in"
)

// PersistentMenuKeyboard gera o teclado persistente 2x2 para a base do chat do Telegram (D-13, D-14).
func PersistentMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(ButtonS2SToday),
			tgbotapi.NewKeyboardButton(ButtonQuickExpense),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(ButtonSimulate),
			tgbotapi.NewKeyboardButton(ButtonCheckin),
		),
	)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = false
	return keyboard
}
