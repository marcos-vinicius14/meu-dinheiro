package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot/fsm"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
)

type pendingExpenseItem struct {
	Amount      float64
	Description string
	CreatedAt   time.Time
}

type Bot struct {
	api             *tgbotapi.BotAPI
	sender          fsm.TelegramSender
	client          *client.APIClient
	logger          *slog.Logger
	cfg             *config.Config
	botName         string
	sessionStore    *fsm.SessionStore
	fsm             *fsm.FSM
	pendingExpenses map[int64]pendingExpenseItem
	pendingMu       sync.RWMutex
}

func NewBot(api *tgbotapi.BotAPI, client *client.APIClient, logger *slog.Logger, cfg *config.Config) *Bot {
	botName := ""
	var sender fsm.TelegramSender
	if api != nil {
		sender = api
		botName = api.Self.UserName
	}
	sessionStore := fsm.NewSessionStore(30 * time.Minute)
	fsmEngine := fsm.NewFSM(sessionStore, sender, client, logger)
	fsmEngine.SetStepHandler(fsm.NewOnboardingHandler(client, logger))
	fsmEngine.SetCheckinHandler(fsm.NewCheckinHandler(client, logger))

	return &Bot{
		api:             api,
		sender:          sender,
		client:          client,
		logger:          logger,
		cfg:             cfg,
		botName:         botName,
		sessionStore:    sessionStore,
		fsm:             fsmEngine,
		pendingExpenses: make(map[int64]pendingExpenseItem),
	}
}

func (b *Bot) SetSender(sender fsm.TelegramSender) {
	b.sender = sender
	b.fsm.SetSender(sender)
}

func (b *Bot) FSM() *fsm.FSM {
	return b.fsm
}

func (b *Bot) SessionStore() *fsm.SessionStore {
	return b.sessionStore
}

func (b *Bot) ProcessUpdate(ctx context.Context, update *tgbotapi.Update) {
	b.handleUpdate(ctx, update)
}

func (b *Bot) Start(ctx context.Context) error {
	go b.sessionStore.StartEvictionWorker(ctx, 5*time.Minute)
	if b.cfg.WebhookURL != "" {
		return b.startWebhook(ctx)
	}
	return b.startPolling(ctx)
}

func (b *Bot) startPolling(ctx context.Context) error {
	if b.api != nil {
		if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
			b.logger.Warn("falha ao resetar webhook anterior para polling", "error", err)
		}
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)
	b.logger.Info("bot iniciado em modo Long Polling", "username", b.botName)

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("encerrando polling do telegram bot")
			b.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			b.handleUpdate(ctx, &update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update *tgbotapi.Update) {
	if update == nil {
		return
	}

	handled, err := b.fsm.HandleUpdate(ctx, update)
	if err != nil {
		b.logger.Error("erro ao processar update na fsm", "error", err)
	}
	if handled {
		return
	}

	if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID
	user := msg.From

	b.logger.Info("mensagem recebida",
		"chat_id", chatID,
		"user_id", user.ID,
		"username", user.UserName,
		"text", text,
	)

	if strings.HasPrefix(text, "/start") {
		args := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if strings.HasPrefix(args, "auth_") {
			token := strings.TrimPrefix(args, "auth_")
			b.handleAuthChallenge(ctx, chatID, user, token)
			return
		}

		welcomeMsg := fmt.Sprintf(
			"👋 Olá, %s!\n\n"+
				"Bem-vindo ao **Meu Dinheiro** — seu gerenciador financeiro pessoal focado no cálculo de Saldo Seguro diário (S2S).\n\n"+
				"Se você estava tentando entrar pelo navegador, utilize o botão de acesso na página para ser redirecionado para cá com seu token de autorização.\n\n"+
				"Comandos disponíveis:\n"+
				"• /s2s - Consultar saldo seguro diário\n"+
				"• /gasto <valor> <descrição> - Lançar gasto rápido\n"+
				"• /renda <valor> <descrição> - Registrar entrada de dinheiro\n"+
				"• /simular <valor> [parcelas] - Simular impacto de compras futuras\n"+
				"• /checkin - Realizar conferência de saldo diária\n"+
				"• /ajuda - Exibir instruções de uso\n"+
				"• /status - Verificar status dos serviços",
			user.FirstName,
		)
		b.replyWithMarkup(chatID, welcomeMsg, PersistentMenuKeyboard())
		return
	}

	// Comandos de Consulta de S2S (D-01, D-15)
	if text == "/s2s" || text == ButtonS2SToday {
		b.handleS2S(ctx, chatID, user.ID)
		return
	}

	// Comandos de Lançamento de Gastos (D-02, D-03, D-04, D-15)
	if text == "/gasto" || text == ButtonQuickExpense {
		b.handleGastoHelp(chatID)
		return
	}

	if strings.HasPrefix(text, "/gasto") {
		b.handleGasto(ctx, chatID, user, text)
		return
	}

	// Comandos de Entrada de Renda/Receita
	if text == "/renda" || text == "/receita" || strings.HasPrefix(text, "/renda ") || strings.HasPrefix(text, "/receita ") {
		b.handleIncomeCommand(ctx, chatID, user, text)
		return
	}

	// Comandos de Simulação What-If (PRED-03, D-05..D-08, D-15)
	if text == "/simular" || text == ButtonSimulate || strings.HasPrefix(text, "/simular ") {
		b.handleSimulateCommand(ctx, chatID, user.ID, text)
		return
	}

	switch text {
	case "/ajuda":
		helpMsg := "📖 **Ajuda - Meu Dinheiro**\n\n" +
			"1. **Saldo Seguro (S2S)**: Utilize `/s2s` ou o botão no teclado para ver quanto pode gastar hoje.\n" +
			"2. **Lançamento de Gastos**: Envie `/gasto <valor> <descrição>` para debitar despesas e ver a recalibração imediata do seu S2S.\n" +
			"3. **Entrada de Dinheiro**: Envie `/renda <valor> <descrição>` para registrar salários, freelas ou recebimentos e aumentar seu S2S na hora.\n" +
			"4. **Simulador What-If**: Envie `/simular <valor> [parcelas]` para simular o impacto de compras em até 12 ciclos futuros.\n" +
			"5. **Check-in Diário**: Envie `/checkin` ou clique no botão para conferir saldo bancário e fechar o dia com chave de ouro.\n" +
			"6. **Login no Navegador**: Acesse a plataforma web e clique em 'Entrar com Telegram'."
		b.replyWithMarkup(chatID, helpMsg, PersistentMenuKeyboard())

	case "/status":
		b.replyWithMarkup(chatID, "🟢 Bot operacional e conectado ao servidor Meu Dinheiro.", PersistentMenuKeyboard())

	default:
		b.replyWithMarkup(chatID, "Comando não reconhecido. Digite /ajuda para ver as instruções.", PersistentMenuKeyboard())
	}
}

func (b *Bot) handleS2S(ctx context.Context, chatID int64, userID int64) {
	userCtx, err := b.client.GetUserContextByTelegram(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			msg := "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*.\n\nEnvie /start para configurar seu perfil financeiro em menos de 2 minutos!"
			b.reply(chatID, msg)
			return
		}
		b.reply(chatID, "❌ Erro ao consultar suas informações financeiras. Tente novamente em instantes.")
		return
	}

	healthBadge := formatHealthBadge(userCtx.Cycle.HealthStatus)
	endDateStr := userCtx.Cycle.EndDate
	if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
		endDateStr = t.Format("02/01/2006")
	}

	// Card Completo de Ciclo (D-01)
	cardMsg := fmt.Sprintf(
		"💰 *Saldo Seguro Diário (S2S):* R$ %s/dia\n"+
			"📊 *Status do Ciclo:* %s\n"+
			"📅 *Dias restantes no ciclo:* %d dias (até %s)\n"+
			"💳 *Saldo Líquido disponível:* R$ %s\n"+
			"📈 *Patrimônio Líquido Total:* R$ %s",
		formatMoneyBR(userCtx.Cycle.S2SToday),
		healthBadge,
		userCtx.Cycle.DaysRemaining,
		endDateStr,
		formatMoneyBR(userCtx.TotalLiquidBalance),
		formatMoneyBR(userCtx.TotalNetWorth),
	)

	if userCtx.Cycle.S2SToday == 0 && (userCtx.Cycle.HealthStatus == "DEFICIT_RISK" || userCtx.Cycle.HealthStatus == "DEFICIT") {
		cardMsg += "\n\n💡 *Dica:* Seu S2S está em R$ 0,00/dia porque suas despesas fixas ou metas de reserva superam o saldo disponível. Registre novas entradas com `/renda <valor> <descrição>` ou ajuste seu saldo em `/checkin`!"
	}

	b.replyWithMarkup(chatID, cardMsg, PersistentMenuKeyboard())
}

func (b *Bot) handleGastoHelp(chatID int64) {
	msg := "💸 *Lançamento Rápido de Gasto*\n\n" +
		"Para registrar uma nova despesa e recalcular seu S2S na hora, envie no formato:\n" +
		"`/gasto <valor> <descrição>`\n\n" +
		"💡 *Exemplos práticos:*\n" +
		"• `/gasto 34.90 Almoço executivo`\n" +
		"• `/gasto 120 Mercado semanal`\n" +
		"• `/gasto 45.00 Farmácia`"
	b.replyWithMarkup(chatID, msg, PersistentMenuKeyboard())
}

func (b *Bot) handleGasto(ctx context.Context, chatID int64, user *tgbotapi.User, text string) {
	args := strings.TrimSpace(strings.TrimPrefix(text, "/gasto"))
	amount, desc, err := fsm.ParseExpenseCommand(args)
	if err != nil {
		b.reply(chatID, fmt.Sprintf("⚠️ %s\n\nExemplo correto: `/gasto 35.00 Almoço`", err.Error()))
		return
	}

	_, err = b.client.GetUserContextByTelegram(ctx, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "não encontrado") {
			b.reply(chatID, "⚠️ Você ainda não possui cadastro no *Meu Dinheiro*. Envie /start para configurar seu perfil.")
			return
		}
		b.reply(chatID, "❌ Erro ao consultar contexto do usuário. Tente novamente.")
		return
	}

	b.pendingMu.Lock()
	b.pendingExpenses[user.ID] = pendingExpenseItem{
		Amount:      amount,
		Description: desc,
		CreatedAt:   time.Now(),
	}
	b.pendingMu.Unlock()

	categories := []string{"Alimentação", "Transporte", "Lazer", "Moradia", "Saúde", "Outros"}
	inlineKeyboard := makeCategoryKeyboard(categories)

	msgText := fmt.Sprintf(
		"🏷️ *Selecione a categoria para esta despesa:*\n"+
			"💸 Valor: *R$ %s*\n"+
			"📝 Descrição: *%s*",
		formatMoneyBR(amount),
		desc,
	)

	b.replyWithMarkup(chatID, msgText, inlineKeyboard)
}

func (b *Bot) handleCallbackQuery(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	if b.sender != nil {
		_, _ = b.sender.Request(tgbotapi.NewCallback(cb.ID, ""))
	}

	chatID := cb.Message.Chat.ID
	user := cb.From

	if cb.Data == "gasto_cancel" {
		b.pendingMu.Lock()
		delete(b.pendingExpenses, user.ID)
		b.pendingMu.Unlock()
		b.replyWithMarkup(chatID, "❌ Lançamento cancelado.", PersistentMenuKeyboard())
		return
	}

	if strings.HasPrefix(cb.Data, "sim_") {
		b.handleSimulateCallback(ctx, cb)
		return
	}

	if strings.HasPrefix(cb.Data, "gasto_cat:") {
		categoryName := strings.TrimPrefix(cb.Data, "gasto_cat:")

		b.pendingMu.Lock()
		pending, exists := b.pendingExpenses[user.ID]
		delete(b.pendingExpenses, user.ID)
		b.pendingMu.Unlock()

		if !exists || time.Since(pending.CreatedAt) > 15*time.Minute {
			b.replyWithMarkup(chatID, "⚠️ Tempo para seleção de categoria expirou. Por favor, envie `/gasto` novamente.", PersistentMenuKeyboard())
			return
		}

		resp, err := b.client.QuickExpense(ctx, client.QuickExpenseRequest{
			TelegramID:   user.ID,
			Amount:       pending.Amount,
			Description:  pending.Description,
			CategoryName: categoryName,
		})
		if err != nil {
			b.logger.Error("erro ao lançar gasto rapido", "error", err)
			b.replyWithMarkup(chatID, "❌ Erro ao registrar despesa na API. Tente novamente.", PersistentMenuKeyboard())
			return
		}

		reduction := resp.PreviousS2S - resp.NewS2S
		healthBadge := formatHealthBadge(resp.HealthStatus)

		// Feedback Comparativo Antes vs Depois (D-02)
		feedbackMsg := fmt.Sprintf(
			"✅ *Gasto de R$ %s registrado em %s!*\n\n"+
				"📉 *S2S Anterior:* R$ %s/dia\n"+
				"💰 *Novo S2S:* R$ %s/dia (*-R$ %s/dia*)\n"+
				"📅 *Dias restantes:* %d dias\n"+
				"📊 *Status do Ciclo:* %s",
			formatMoneyBR(pending.Amount),
			categoryName,
			formatMoneyBR(resp.PreviousS2S),
			formatMoneyBR(resp.NewS2S),
			formatMoneyBR(reduction),
			resp.DaysRemaining,
			healthBadge,
		)

		b.replyWithMarkup(chatID, feedbackMsg, PersistentMenuKeyboard())
		return
	}
}

func formatMoneyBR(val float64) string {
	s := fmt.Sprintf("%.2f", val)
	return strings.Replace(s, ".", ",", 1)
}

func makeCategoryKeyboard(categories []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for _, cat := range categories {
		btn := tgbotapi.NewInlineKeyboardButtonData(cat, "gasto_cat:"+cat)
		currentRow = append(currentRow, btn)
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	cancelRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancelar", "gasto_cancel"),
	}
	rows = append(rows, cancelRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func formatHealthBadge(status string) string {
	switch status {
	case "HEALTHY":
		return "🟢 SAUDÁVEL"
	case "RESTRICTED":
		return "🟡 RESTRITO"
	case "ATTENTION":
		return "🟡 ATENÇÃO"
	case "DEFICIT":
		return "🔴 DÉFICIT"
	case "DEFICIT_RISK", "DEFICIT_WARNING":
		return "🔴 RISCO DE DÉFICIT"
	default:
		clean := strings.ReplaceAll(status, "_", " ")
		if clean != "" {
			return clean
		}
		return "🟢 SAUDÁVEL"
	}
}

func (b *Bot) handleAuthChallenge(ctx context.Context, chatID int64, user *tgbotapi.User, token string) {
	name := user.FirstName
	if user.LastName != "" {
		name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	}

	b.logger.Info("autorizando challenge de login", "telegram_id", user.ID, "token_prefix", token[:min(8, len(token))])

	err := b.client.AuthorizeChallenge(ctx, token, user.ID, name)
	if err != nil {
		b.logger.Error("falha ao autorizar challenge", "error", err)
		msg := "❌ **Não foi possível autorizar o login.**\n\n" +
			"O link de acesso pode ter expirado ou já ter sido utilizado.\n" +
			"Por favor, volte ao navegador e gere um novo link de login."
		b.reply(chatID, msg)
		return
	}

	successMsg := "✅ **Login autorizado com sucesso!**\n\n" +
		"Seu acesso ao Meu Dinheiro foi liberado no navegador.\n" +
		"Você já pode fechar o Telegram e continuar na página web."
	b.reply(chatID, successMsg)
}

func (b *Bot) send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if b.sender != nil {
		return b.sender.Send(c)
	}
	if b.api != nil {
		return b.api.Send(c)
	}
	return tgbotapi.Message{}, nil
}

func (b *Bot) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := b.send(msg); err != nil {
		b.logger.Error("erro ao enviar mensagem telegram", "chat_id", chatID, "error", err)
	}
}

func (b *Bot) replyWithMarkup(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup
	if _, err := b.send(msg); err != nil {
		b.logger.Error("erro ao enviar mensagem telegram com markup", "chat_id", chatID, "error", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
