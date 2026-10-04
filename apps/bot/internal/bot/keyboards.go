package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
)

const (
	ButtonS2SToday     = fsm.ButtonS2SToday
	ButtonQuickExpense = fsm.ButtonQuickExpense
	ButtonSimulate     = fsm.ButtonSimulate
	ButtonCheckin      = fsm.ButtonCheckin
)

// PersistentMenuKeyboard reexporta o teclado persistente 2x2 do pacote fsm (D-13, D-14).
func PersistentMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return fsm.PersistentMenuKeyboard()
}
