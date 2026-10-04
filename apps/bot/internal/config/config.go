package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	TelegramBotToken   string
	APIBaseURL         string
	InternalAPIKey     string
	WebhookURL         string
	WebhookSecretToken string
	WebhookPort        string
	WebhookPath        string
	WebhookCertPath    string
	WebhookKeyPath     string
}

func Load() *Config {
	loadEnv()

	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8081"
		}
		apiBaseURL = fmt.Sprintf("http://api:%s", port)
	}

	internalAPIKey := os.Getenv("INTERNAL_API_KEY")
	if internalAPIKey == "" {
		internalAPIKey = "dev-internal-api-key-bot"
	}

	webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_URL"))
	webhookSecretToken := strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TOKEN"))

	webhookPort := ""
	if webhookURL != "" {
		if parsed, err := url.Parse(webhookURL); err == nil && parsed.Port() != "" {
			webhookPort = parsed.Port()
		}
	}
	if webhookPort == "" {
		webhookPort = strings.TrimSpace(os.Getenv("WEBHOOK_PORT"))
	}
	if webhookPort == "" {
		webhookPort = strings.TrimSpace(os.Getenv("PORT"))
	}
	if webhookPort == "" {
		webhookPort = "8443"
	}

	webhookPath := strings.TrimSpace(os.Getenv("WEBHOOK_PATH"))
	if webhookPath == "" && webhookURL != "" {
		if parsed, err := url.Parse(webhookURL); err == nil && parsed.Path != "" && parsed.Path != "/" {
			webhookPath = parsed.Path
		}
	}
	if webhookPath == "" {
		webhookPath = "/webhook"
	}

	// Normaliza a WEBHOOK_URL para garantir que inclua o webhookPath
	if webhookURL != "" {
		if parsed, err := url.Parse(webhookURL); err == nil {
			if parsed.Path == "" || parsed.Path == "/" {
				parsed.Path = webhookPath
				webhookURL = parsed.String()
			}
		}
	}

	return &Config{
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		APIBaseURL:         apiBaseURL,
		InternalAPIKey:     internalAPIKey,
		WebhookURL:         webhookURL,
		WebhookSecretToken: webhookSecretToken,
		WebhookPort:        webhookPort,
		WebhookPath:        webhookPath,
		WebhookCertPath:    strings.TrimSpace(os.Getenv("WEBHOOK_CERT_PATH")),
		WebhookKeyPath:     strings.TrimSpace(os.Getenv("WEBHOOK_KEY_PATH")),
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
