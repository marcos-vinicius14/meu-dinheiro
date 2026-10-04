package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	Port           string
	JWTSecret      string
	BotUsername    string
	InternalAPIKey string
}

func Load() *Config {
	loadEnv()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:altere_para_uma_senha_forte_em_producao@postgres:5432/meu_dinheiro?sslmode=disable"
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

func loadEnv() {
	loadFromFile := func(filename string) bool {
		paths := []string{filename, "../../" + filename, "../" + filename}
		for _, p := range paths {
			f, err := os.Open(p)
			if err != nil {
				continue
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if _, exists := os.LookupEnv(key); !exists {
						_ = os.Setenv(key, val)
					}
				}
			}
			return true
		}
		return false
	}

	// Carrega .env.local prioritariamente; se não existir, tenta .env
	if loadFromFile(".env.local") {
		return
	}
	loadFromFile(".env")
}
