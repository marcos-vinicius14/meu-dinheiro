package config

import (
	"os"
)

type Config struct {
	TelegramBotToken string
	APIBaseURL       string
	InternalAPIKey   string
}

func Load() *Config {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}

	internalAPIKey := os.Getenv("INTERNAL_API_KEY")
	if internalAPIKey == "" {
		internalAPIKey = "dev-internal-api-key-bot"
	}

	return &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		APIBaseURL:       apiBaseURL,
		InternalAPIKey:   internalAPIKey,
	}
}
