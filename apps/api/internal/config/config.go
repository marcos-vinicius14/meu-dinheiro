package config

import (
	"os"
)

type Config struct {
	DatabaseURL    string
	Port           string
	JWTSecret      string
	BotUsername    string
	InternalAPIKey string
}

func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/meudinheiro?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("SERVER_PORT")
	}
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-meu-dinheiro-super-secure-key-2026"
	}

	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		botUsername = "MeuDinheiroBot"
	}

	internalAPIKey := os.Getenv("INTERNAL_API_KEY")
	if internalAPIKey == "" {
		internalAPIKey = "dev-internal-api-key-bot"
	}

	return &Config{
		DatabaseURL:    dbURL,
		Port:           port,
		JWTSecret:      jwtSecret,
		BotUsername:    botUsername,
		InternalAPIKey: internalAPIKey,
	}
}
