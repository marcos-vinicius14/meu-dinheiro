package fsm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

type CheckinHandler struct {
	client *client.APIClient
	logger *slog.Logger
}

func NewCheckinHandler(client *client.APIClient, logger *slog.Logger) *CheckinHandler {
	return &CheckinHandler{
		client: client,
		logger: logger,
	}
}

// StartCheckin inicia o diálogo guiado de check-in diário (D-09, D-15).
func (h *CheckinHandler) StartCheckin(ctx context.Context, f *FSM, sess *Session) error {
	userCtx, err := h.client.GetUserContextByTelegram(ctx, sess.TelegramID)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			f.Reply(sess.ChatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil em 2 minutos!")
			return nil
		}
		f.Reply(sess.ChatID, "❌ Erro ao consultar contexto financeiro. Tente novamente mais tarde.")
		return nil
	}

	sess.CheckinCalculatedLiquid = userCtx.TotalLiquidBalance
	sess.CheckinExpenses = nil
	sess.CurrentState = StateCheckinWaitingExpenses
	f.store.Touch(sess.TelegramID)

	msg := "📝 *Check-in Diário — Conferência de Gastos*\n\n" +
		"Hoje você teve alguma despesa que ainda não registrou no bot?"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Sim, lançar", string(ActionCheckinAddExpense)),
			tgbotapi.NewInlineKeyboardButtonData("👍 Não, tudo certo", string(ActionCheckinNoExpenses)),
		),
	)

	f.ReplyWithKeyboard(sess.ChatID, msg, keyboard)
	return nil
}

func (h *CheckinHandler) HandleStepCallback(ctx context.Context, f *FSM, sess *Session, data string) error {
	switch Action(data) {
	case ActionCheckinAddExpense:
		sess.CurrentState = StateCheckinEnteringExpense
		f.store.Touch(sess.TelegramID)
		f.Reply(sess.ChatID, "Envie a despesa no formato: `<valor> <descrição>` (ex: `35 Almoço`):")
		return nil

	case ActionCheckinNoExpenses, ActionCheckinFinishExpenses:
		return h.promptBalanceConfirm(f, sess)

	case ActionCheckinBalanceOK:
		return h.finishCheckin(ctx, f, sess)

	case ActionCheckinBalanceAdjust:
		sess.CurrentState = StateCheckinEnteringBalance
		f.store.Touch(sess.TelegramID)
		f.Reply(sess.ChatID, "Informe o saldo real atual somando suas contas correntes (ex: `4100.00` ou `4.100,00`):")
		return nil

	default:
		return nil
	}
}

func (h *CheckinHandler) HandleStepMessage(ctx context.Context, f *FSM, sess *Session, text string) error {
	switch sess.CurrentState {
	case StateCheckinEnteringExpense:
		amount, desc, err := ParseExpenseCommand(text)
		if err != nil {
			f.Reply(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nEnvie no formato: `<valor> <descrição>` (ex: `35 Almoço`):", err.Error()))
			return nil
		}

		categoryName := inferCategoryName(desc)
		sess.CheckinExpenses = append(sess.CheckinExpenses, FixedExpenseData{
			Description:  desc,
			Amount:       amount,
			CategoryName: categoryName,
		})
		f.store.Touch(sess.TelegramID)

		totalAdded := 0.0
		for _, e := range sess.CheckinExpenses {
			totalAdded += e.Amount
		}

		msg := fmt.Sprintf(
			"✅ Item adicionado: *%s* (R$ %s) [%s].\nTotal em itens novos: *R$ %s*.\n\n"+
				"Envie outro gasto ou clique abaixo para prosseguir com a conferência de saldo:",
			desc,
			formatMoneyBR(amount),
			categoryName,
			formatMoneyBR(totalAdded),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Concluir Despesas", string(ActionCheckinFinishExpenses)),
			),
		)

		f.ReplyWithKeyboard(sess.ChatID, msg, keyboard)
		return nil

	case StateCheckinEnteringBalance:
		val, err := ParseMoney(text)
		if err != nil {
			f.Reply(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nPor favor, informe seu saldo real atual (ex: `4100.00`):", err.Error()))
			return nil
		}

		_, err = h.client.AdjustAccountBalance(ctx, client.AdjustBalanceRequest{
			TelegramID: sess.TelegramID,
			NewBalance: val,
		})
		if err != nil {
			h.logger.Error("erro ao ajustar saldo no checkin", "error", err)
			f.Reply(sess.ChatID, "⚠️ Erro ao atualizar saldo bancário na API. Prosseguindo com o check-in...")
		} else {
			sess.CheckinCalculatedLiquid = val
		}

		return h.finishCheckin(ctx, f, sess)

	default:
		return nil
	}
}

func (h *CheckinHandler) promptBalanceConfirm(f *FSM, sess *Session) error {
	sess.CurrentState = StateCheckinWaitingBalanceConfirm
	f.store.Touch(sess.TelegramID)

	msg := fmt.Sprintf(
		"🏦 *Conferência de Saldo Bancário:*\n\n"+
			"Seu saldo líquido calculado em conta é de *R$ %s*.\n\n"+
			"Esse valor confere com a soma das suas contas hoje?",
		formatMoneyBR(sess.CheckinCalculatedLiquid),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Sim, confere", string(ActionCheckinBalanceOK)),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Ajustar saldo", string(ActionCheckinBalanceAdjust)),
		),
	)

	f.ReplyWithKeyboard(sess.ChatID, msg, keyboard)
	return nil
}

func (h *CheckinHandler) finishCheckin(ctx context.Context, f *FSM, sess *Session) error {
	untracked := make([]client.CheckInExpenseInput, len(sess.CheckinExpenses))
	for i, e := range sess.CheckinExpenses {
		untracked[i] = client.CheckInExpenseInput{
			Description:  e.Description,
			Amount:       e.Amount,
			CategoryName: e.CategoryName,
		}
	}

	resp, err := h.client.DailyCheckIn(ctx, client.DailyCheckInRequest{
		TelegramID:        sess.TelegramID,
		UntrackedExpenses: untracked,
	})
	if err != nil {
		h.logger.Error("falha ao enviar check-in para api", "error", err)
		f.ReplyWithReplyMarkup(sess.ChatID, "❌ Erro ao processar check-in diário na API. Tente novamente em instantes.", PersistentMenuKeyboard())
		return nil
	}

	healthBadge := formatHealthBadge(resp.HealthStatus)
	diffTomorrow := resp.NextDayS2S - resp.S2SCalculated
	diffSignal := "+"
	if diffTomorrow < 0 {
		diffSignal = ""
	}

	// Relatório de Conquista Diária (D-11)
	reportMsg := fmt.Sprintf(
		"🏁 *Check-in Diário Concluído!*\n\n"+
			"📊 *Resumo de Hoje:*\n"+
			"• Gastos realizados: *R$ %s*\n"+
			"• Cota do dia (S2S): *R$ %s*\n"+
			"🎉 *Você economizou R$ %s hoje!*\n\n"+
			"💰 *S2S Projetado para Amanhã:* R$ %s/dia (*%sR$ %s/dia*)\n"+
			"%s Ciclo segue %s. Até amanhã!",
		formatMoneyBR(resp.SpentToday),
		formatMoneyBR(resp.DailyQuota),
		formatMoneyBR(resp.DeltaSavings),
		formatMoneyBR(resp.NextDayS2S),
		diffSignal,
		formatMoneyBR(diffTomorrow),
		healthBadge,
		resp.HealthStatus,
	)

	sess.CurrentState = StateIdle
	sess.CheckinExpenses = nil
	f.store.Touch(sess.TelegramID)

	f.ReplyWithReplyMarkup(sess.ChatID, reportMsg, PersistentMenuKeyboard())
	return nil
}

func formatMoneyBR(val float64) string {
	s := fmt.Sprintf("%.2f", val)
	return strings.Replace(s, ".", ",", 1)
}
