package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	client  *client.APIClient
	logger  *slog.Logger
	cfg     *config.Config
	botName string
}

func NewBot(api *tgbotapi.BotAPI, client *client.APIClient, logger *slog.Logger, cfg *config.Config) *Bot {
	botName := ""
	if api != nil {
		botName = api.Self.UserName
	}
	return &Bot{
		api:     api,
		client:  client,
		logger:  logger,
		cfg:     cfg,
		botName: botName,
	}
}

func (b *Bot) Start(ctx context.Context) error {
	if b.cfg.WebhookURL != "" {
		return b.startWebhook(ctx)
	}
	return b.startPolling(ctx)
}

func (b *Bot) startPolling(ctx context.Context) error {
	// Remove qualquer webhook ativo para garantir que o Long Polling funcione sem conflitos no Telegram
	if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
		b.logger.Warn("falha ao resetar webhook anterior para polling", "error", err)
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
	if update == nil || update.Message == nil {
		return
	}
	b.handleMessage(ctx, update.Message)
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
				"• /ajuda - Exibir instruções de uso\n"+
				"• /status - Verificar status dos serviços",
			user.FirstName,
		)
		b.reply(chatID, welcomeMsg)
		return
	}

	switch text {
	case "/ajuda":
		helpMsg := "📖 **Ajuda - Meu Dinheiro**\n\n" +
			"1. **Login no Navegador**: Acesse a plataforma web e clique em 'Entrar com Telegram'. O link abrirá esta conversa com seu token de acesso seguro.\n" +
			"2. **Saldo Seguro (S2S)**: Acompanhe seus gastos e limites projetados para nunca fechar o ciclo no vermelho."
		b.reply(chatID, helpMsg)

	case "/status":
		b.reply(chatID, "🟢 Bot operacional e conectado ao servidor Meu Dinheiro.")

	default:
		b.reply(chatID, "Comando não reconhecido. Digite /ajuda para ver as instruções.")
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

func (b *Bot) reply(chatID int64, text string) {
	if b.api == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("erro ao enviar mensagem telegram", "chat_id", chatID, "error", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
