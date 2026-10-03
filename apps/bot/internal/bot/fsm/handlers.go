package fsm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
)

type OnboardingHandler struct {
	client *client.APIClient
	logger *slog.Logger
}

func NewOnboardingHandler(client *client.APIClient, logger *slog.Logger) *OnboardingHandler {
	return &OnboardingHandler{
		client: client,
		logger: logger,
	}
}

func (h *OnboardingHandler) HandleStart(ctx context.Context, f *FSM, sess *Session) error {
	userCtx, err := h.client.GetUserContextByTelegram(ctx, sess.TelegramID)
	if err == nil && userCtx != nil && userCtx.User.ID != "" {
		sess.CurrentState = StateIdle

		healthBadge := formatHealthBadge(userCtx.Cycle.HealthStatus)
		endDateStr := userCtx.Cycle.EndDate
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDateStr = t.Format("02/01/2006")
		}

		dashboardMsg := fmt.Sprintf(
			"👋 Olá, *%s*! Bem-vindo de volta ao *Meu Dinheiro*.\n\n"+
				"📊 *Painel do Dia:*\n"+
				"• *S2S de Hoje:* R$ %.2f/dia\n"+
				"• *Status:* %s\n"+
				"• *Dias Restantes no Ciclo:* %d dias (até %s)\n"+
				"• *Saldo Líquido:* R$ %.2f\n"+
				"• *Reserva de Emergência:* %.1f meses cobertos (%.1f%%)\n"+
				"• *Patrimônio Líquido:* R$ %.2f\n\n"+
				"⚡ *Comandos Rápidos:*\n"+
				"• `/s2s` — Consultar saldo seguro diário\n"+
				"• `/gasto <valor> <descrição>` — Registrar despesa rápida\n"+
				"• `/status` — Visão geral da conta\n"+
				"• `/ajuda` — Todos os comandos",
			sess.FirstName,
			userCtx.Cycle.S2SToday,
			healthBadge,
			userCtx.Cycle.DaysRemaining,
			endDateStr,
			userCtx.TotalLiquidBalance,
			userCtx.EmergencyFund.MonthsCovered,
			userCtx.EmergencyFund.ProgressPercent,
			userCtx.TotalNetWorth,
		)
		f.Reply(sess.ChatID, dashboardMsg)
		return nil
	}

	sess.CurrentState = StateWaitingBalance
	welcomeMsg := fmt.Sprintf(
		"👋 Olá, *%s*! Bem-vindo ao *Meu Dinheiro*.\n\n"+
			"Aqui você gerencia suas finanças com o método do **S2S (Saldo Seguro Diário)**, que calcula exatamente quanto você pode gastar por dia sem fechar o ciclo no vermelho.\n\n"+
			"Vamos fazer uma rápida configuração inicial (leva menos de 2 minutos)!\n\n"+
			"👉 *Pergunta 1 de 5:*\n"+
			"Para começar, qual o seu **saldo total** somando suas contas correntes hoje? (Ex: `3500.00` ou `3.500,00`)",
		sess.FirstName,
	)
	f.Reply(sess.ChatID, welcomeMsg)
	return nil
}

func (h *OnboardingHandler) HandleStepMessage(ctx context.Context, f *FSM, sess *Session, text string) error {
	switch sess.CurrentState {
	case StateWaitingBalance:
		return h.handleStepBalance(ctx, f, sess, text)
	case StateWaitingCycleDay:
		return h.handleStepCycleDay(ctx, f, sess, text)
	case StateWaitingFixedExpenses:
		return h.handleStepFixedExpenses(ctx, f, sess, text)
	case StateWaitingSavingsTarget:
		return h.handleStepSavingsTarget(ctx, f, sess, text)
	case StateWaitingInvestmentInput:
		return h.handleStepInvestmentInput(ctx, f, sess, text)
	default:
		return nil
	}
}

func (h *OnboardingHandler) handleStepBalance(ctx context.Context, f *FSM, sess *Session, text string) error {
	val, err := ParseMoney(text)
	if err != nil {
		f.Reply(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nPor favor, informe seu saldo total em conta corrente (ex: `3500.00`):", err.Error()))
		return nil
	}
	sess.InitialBalance = val
	sess.CurrentState = StateWaitingCycleDay

	msg := fmt.Sprintf(
		"🗓️ Saldo de **R$ %.2f** registrado com sucesso!\n\n"+
			"👉 *Pergunta 2 de 5:*\n"+
			"Em qual dia costuma cair seu salário para reiniciarmos seu ciclo mensal? (Padrão: `01`, limite máximo: `28`):",
		val,
	)
	f.Reply(sess.ChatID, msg)
	return nil
}

func (h *OnboardingHandler) handleStepCycleDay(ctx context.Context, f *FSM, sess *Session, text string) error {
	day, err := ParseCycleDay(text)
	if err != nil {
		f.Reply(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nPor favor, envie um dia válido entre 1 e 28:", err.Error()))
		return nil
	}
	sess.CycleStartDay = day
	sess.CurrentState = StateWaitingFixedExpenses

	msg := fmt.Sprintf(
		"✅ Ciclo definido para o dia **%02d** de cada mês!\n\n"+
			"👉 *Pergunta 3 de 5:*\n"+
			"Quanto você estima gastar por mês com **despesas essenciais** (como aluguel, luz, condomínio, mercado)?\n\n"+
			"Envie uma por mensagem (ex: `Aluguel 1500` ou `Luz 250,00`).\n\n"+
			"Quando terminar de cadastrar todas as contas essenciais, clique no botão abaixo:",
		day,
	)
	f.ReplyWithKeyboard(sess.ChatID, msg, makeFinishFixedExpensesKeyboard())
	return nil
}

func (h *OnboardingHandler) handleStepFixedExpenses(ctx context.Context, f *FSM, sess *Session, text string) error {
	exp, err := ParseFixedExpense(text)
	if err != nil {
		f.ReplyWithKeyboard(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nEnvie no formato `Descrição Valor` (ex: `Aluguel 1500`) ou conclua:", err.Error()), makeFinishFixedExpensesKeyboard())
		return nil
	}
	sess.FixedExpenses = append(sess.FixedExpenses, *exp)
	sess.MonthlyEssentialCost += exp.Amount

	msg := fmt.Sprintf(
		"➕ Adicionado: **%s** (R$ %.2f) em *%s*.\nTotal essencial acumulado: **R$ %.2f/mês**.\n\n"+
			"Envie mais uma despesa ou clique abaixo para concluir:",
		exp.Description,
		exp.Amount,
		exp.CategoryName,
		sess.MonthlyEssentialCost,
	)
	f.ReplyWithKeyboard(sess.ChatID, msg, makeFinishFixedExpensesKeyboard())
	return nil
}

func (h *OnboardingHandler) handleStepSavingsTarget(ctx context.Context, f *FSM, sess *Session, text string) error {
	val, err := ParseMoney(text)
	if err != nil {
		f.Reply(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nQuanto você planeja guardar por mês para essa reserva? (Ex: `300.00`):", err.Error()))
		return nil
	}
	sess.TargetSavings = val
	sess.CurrentState = StateWaitingInvestmentChoice

	msg := fmt.Sprintf(
		"💰 Meta de aporte de **R$ %.2f/mês** registrada!\n\n"+
			"👉 *Pergunta 5 de 5:*\n"+
			"Você possui dinheiro investido em ações, FIIs ou outros ativos que gostaria de cadastrar agora?",
		val,
	)
	f.ReplyWithKeyboard(sess.ChatID, msg, makeInvestmentChoiceKeyboard())
	return nil
}

func (h *OnboardingHandler) handleStepInvestmentInput(ctx context.Context, f *FSM, sess *Session, text string) error {
	inv, err := ParseInvestment(text)
	if err != nil {
		f.ReplyWithKeyboard(sess.ChatID, fmt.Sprintf("⚠️ %s\n\nEnvie no formato: `TICKER QUANTIDADE PREÇO` (ex: `ALUP11 10 42.23`) ou conclua:", err.Error()), makeAfterInvestmentKeyboard())
		return nil
	}
	sess.Investments = append(sess.Investments, *inv)

	msg := fmt.Sprintf(
		"📈 Ativo **%s** cadastrado com sucesso! (%.2f cotas a R$ %.2f cada).\n\n"+
			"Adicione outro ativo ou finalize o onboarding:",
		inv.Ticker,
		inv.Quantity,
		inv.AveragePrice,
	)
	f.ReplyWithKeyboard(sess.ChatID, msg, makeAfterInvestmentKeyboard())
	return nil
}

func (h *OnboardingHandler) HandleStepCallback(ctx context.Context, f *FSM, sess *Session, data string) error {
	action := Action(data)

	switch sess.CurrentState {
	case StateWaitingFixedExpenses:
		if action == ActionFinishFixedExpenses {
			return h.onFinishFixedExpenses(ctx, f, sess)
		}

	case StateWaitingEmergencyFundChoice:
		switch action {
		case ActionFund6:
			return h.onChooseEmergencyFund(ctx, f, sess, 6)
		case ActionFund12:
			return h.onChooseEmergencyFund(ctx, f, sess, 12)
		}

	case StateWaitingInvestmentChoice:
		switch action {
		case ActionAddInvestment:
			return h.onPromptInvestmentInput(ctx, f, sess)
		case ActionSkipInvestments:
			return h.finalizeOnboarding(ctx, f, sess)
		}

	case StateWaitingInvestmentInput:
		switch action {
		case ActionAddInvestment:
			return h.onPromptInvestmentInput(ctx, f, sess)
		case ActionFinishInvestments:
			return h.finalizeOnboarding(ctx, f, sess)
		}
	}

	h.logger.Warn("callback recebido em estado incompativel",
		"telegram_id", sess.TelegramID,
		"current_state", sess.CurrentState,
		"action", action,
	)
	f.Reply(sess.ChatID, "⚠️ Essa opção pertence a uma etapa anterior ou não é mais válida. Por favor, responda à última pergunta.")
	return nil
}

func (h *OnboardingHandler) onFinishFixedExpenses(ctx context.Context, f *FSM, sess *Session) error {
	sess.CurrentState = StateWaitingEmergencyFundChoice
	msg := fmt.Sprintf(
		"🛡️ Seu custo essencial mensal é de **R$ %.2f**.\n\n"+
			"👉 *Pergunta 4 de 5:*\n"+
			"Para sua segurança financeira, o método recomenda montar uma Reserva de Emergência.\n\n"+
			"Você prefere uma meta de **6 meses (CLT)** ou **12 meses (PJ/Autônomo)**?",
		sess.MonthlyEssentialCost,
	)
	f.ReplyWithKeyboard(sess.ChatID, msg, makeEmergencyFundKeyboard(sess.MonthlyEssentialCost))
	return nil
}

func (h *OnboardingHandler) onChooseEmergencyFund(ctx context.Context, f *FSM, sess *Session, months int) error {
	sess.EmergencyFundMonths = months
	sess.CurrentState = StateWaitingSavingsTarget

	var label, example string
	if months == 6 {
		label = "6 meses (CLT)"
		example = "300.00"
	} else {
		label = "12 meses (PJ/Autônomo)"
		example = "500.00"
	}

	msg := fmt.Sprintf(
		"🎯 Meta de **%s** selecionada!\n\n"+
			"Quanto você planeja guardar por mês para construir essa reserva? (Ex: `%s`):",
		label,
		example,
	)
	f.Reply(sess.ChatID, msg)
	return nil
}

func (h *OnboardingHandler) onPromptInvestmentInput(ctx context.Context, f *FSM, sess *Session) error {
	sess.CurrentState = StateWaitingInvestmentInput
	msg := "📈 Envie os dados do ativo no formato:\n`TICKER QUANTIDADE PREÇO` (ex: `ALUP11 10 42.23` ou `PETR4 100 38.50`):"
	f.Reply(sess.ChatID, msg)
	return nil
}

func (h *OnboardingHandler) finalizeOnboarding(ctx context.Context, f *FSM, sess *Session) error {
	var usernamePtr *string
	if sess.Username != "" {
		u := sess.Username
		usernamePtr = &u
	}

	var fixedExpenses []client.FixedExpenseInput
	for _, fe := range sess.FixedExpenses {
		fixedExpenses = append(fixedExpenses, client.FixedExpenseInput{
			Description:  fe.Description,
			Amount:       fe.Amount,
			CategoryName: fe.CategoryName,
		})
	}

	var investments []client.InitialInvestmentInput
	for _, inv := range sess.Investments {
		investments = append(investments, client.InitialInvestmentInput{
			Ticker:       inv.Ticker,
			Quantity:     inv.Quantity,
			AveragePrice: inv.AveragePrice,
		})
	}

	fundMonths := sess.EmergencyFundMonths
	if fundMonths != 6 && fundMonths != 12 {
		fundMonths = 6
	}

	cycleDay := sess.CycleStartDay
	if cycleDay < 1 || cycleDay > 28 {
		cycleDay = 1
	}

	req := client.OnboardingRequest{
		TelegramID:          sess.TelegramID,
		FirstName:           sess.FirstName,
		Username:            usernamePtr,
		InitialBalance:      sess.InitialBalance,
		CycleStartDay:       cycleDay,
		FixedExpenses:       fixedExpenses,
		EmergencyFundMonths: fundMonths,
		TargetSavings:       sess.TargetSavings,
		Investments:         investments,
	}

	resp, err := h.client.SaveOnboarding(ctx, req)
	if err != nil {
		h.logger.Error("falha ao salvar onboarding na api", "telegram_id", sess.TelegramID, "error", err)
		f.Reply(sess.ChatID, "❌ Ocorreu um erro ao salvar suas configurações. Tente novamente ou use /cancelar.")
		return err
	}

	// Remove sessão em memória ao concluir com sucesso
	f.GetStore().Delete(sess.TelegramID)

	healthBadge := formatHealthBadge(resp.Cycle.HealthStatus)
	endDateStr := resp.Cycle.EndDate
	if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
		endDateStr = t.Format("02/01/2006")
	}

	reportMsg := fmt.Sprintf(
		"🎉 *Configuração Concluída com Sucesso!*\n\n"+
			"📊 *Seu Painel Financeiro Inicial:*\n"+
			"• *S2S (Saldo Seguro Diário):* R$ %.2f/dia\n"+
			"• *Status do Ciclo:* %s\n"+
			"• *Dias Restantes no Ciclo:* %d dias (até %s)\n"+
			"• *Saldo Líquido em Conta:* R$ %.2f\n"+
			"• *Reserva de Emergência:* R$ %.2f (Meta: R$ %.2f — %.1f%%)\n"+
			"• *Investimentos:* R$ %.2f\n"+
			"• *Patrimônio Líquido Total:* R$ %.2f\n\n"+
			"💡 *Como usar seu bot no dia a dia:*\n"+
			"• `/s2s` — Consulte seu saldo seguro a qualquer momento\n"+
			"• `/gasto 25.50 Almoço` — Registre saídas rápidas\n"+
			"• `/status` — Resumo do seu ciclo atual\n"+
			"• `/ajuda` — Lista completa de comandos",
		resp.Cycle.S2SToday,
		healthBadge,
		resp.Cycle.DaysRemaining,
		endDateStr,
		resp.TotalLiquidBalance,
		resp.TotalLiquidBalance,
		resp.EmergencyFund.ChosenTarget,
		resp.EmergencyFund.ProgressPercent,
		resp.TotalInvested,
		resp.TotalNetWorth,
	)

	f.Reply(sess.ChatID, reportMsg)
	return nil
}

const (
	HealthStatusHealthy        = "HEALTHY"
	HealthStatusRestricted     = "RESTRICTED"
	HealthStatusDeficitRisk    = "DEFICIT_RISK"
	HealthStatusDeficitWarning = "DEFICIT_WARNING"
)

func formatHealthBadge(status string) string {
	switch status {
	case HealthStatusHealthy:
		return "🟢 SAUDÁVEL"
	case HealthStatusRestricted:
		return "🟡 RESTRITO"
	case HealthStatusDeficitRisk, HealthStatusDeficitWarning:
		return "🔴 RISCO DE DÉFICIT"
	default:
		clean := strings.ReplaceAll(status, "_", " ")
		if clean != "" {
			return clean
		}
		return "🟢 SAUDÁVEL"
	}
}

func makeFinishFixedExpensesKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Concluir Despesas Fixas", string(ActionFinishFixedExpenses)),
		),
	)
}

func makeEmergencyFundKeyboard(cost float64) tgbotapi.InlineKeyboardMarkup {
	s6 := cost * 6
	s12 := cost * 12
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🎯 6 Meses (CLT) - R$ %.2f", s6), string(ActionFund6)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🎯 12 Meses (PJ) - R$ %.2f", s12), string(ActionFund12)),
		),
	)
}

func makeInvestmentChoiceKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Adicionar Ativo", string(ActionAddInvestment)),
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Pular Etapa", string(ActionSkipInvestments)),
		),
	)
}

func makeAfterInvestmentKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Adicionar Outro Ativo", string(ActionAddInvestment)),
			tgbotapi.NewInlineKeyboardButtonData("✅ Finalizar Onboarding", string(ActionFinishInvestments)),
		),
	)
}
