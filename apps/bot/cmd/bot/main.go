package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	if cfg.TelegramBotToken == "" {
		logger.Error("TELEGRAM_BOT_TOKEN e obrigatorio")
		os.Exit(1)
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		logger.Error("falha ao inicializar telegram bot api", "error", err)
		os.Exit(1)
	}

	apiClient := client.NewAPIClient(cfg.APIBaseURL, cfg.InternalAPIKey)
	b := bot.NewBot(botAPI, apiClient, logger, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info("iniciando telegram bot...", "bot_user", botAPI.Self.UserName)
	if err := b.Start(ctx); err != nil {
		logger.Error("erro no telegram bot", "error", err)
		os.Exit(1)
	}
	logger.Info("telegram bot finalizado")
}
