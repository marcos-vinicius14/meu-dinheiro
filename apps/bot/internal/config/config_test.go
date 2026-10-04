package config_test

import (
	"os"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Limpa variáveis de teste para garantir isolamento
	origPort := os.Getenv("PORT")
	os.Unsetenv("PORT")
	defer func() {
		if origPort != "" {
			_ = os.Setenv("PORT", origPort)
		}
	}()

	os.Unsetenv("WEBHOOK_URL")
	os.Unsetenv("WEBHOOK_SECRET_TOKEN")
	os.Unsetenv("WEBHOOK_PORT")
	os.Unsetenv("WEBHOOK_PATH")

	cfg := config.Load()
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.WebhookURL)
	assert.Empty(t, cfg.WebhookSecretToken)
	assert.Equal(t, "8443", cfg.WebhookPort)
	assert.Equal(t, "/webhook", cfg.WebhookPath)
	assert.Equal(t, "http://api:8081", cfg.APIBaseURL)
}

func TestConfigLoadWithWebhook(t *testing.T) {
	t.Setenv("WEBHOOK_URL", "https://bot.example.com/custom-webhook")
	t.Setenv("WEBHOOK_SECRET_TOKEN", "super-secret-123456")
	t.Setenv("WEBHOOK_PORT", "9090")

	cfg := config.Load()
	assert.Equal(t, "https://bot.example.com/custom-webhook", cfg.WebhookURL)
	assert.Equal(t, "super-secret-123456", cfg.WebhookSecretToken)
	assert.Equal(t, "9090", cfg.WebhookPort)
	assert.Equal(t, "/custom-webhook", cfg.WebhookPath)
}

func TestConfigLoadWithWebhookURLWithoutPath(t *testing.T) {
	t.Setenv("WEBHOOK_URL", "https://bot.example.com")
	t.Setenv("WEBHOOK_SECRET_TOKEN", "token-abc")
	t.Setenv("PORT", "8888")

	cfg := config.Load()
	// Deve normalizar adicionando /webhook
	assert.Equal(t, "https://bot.example.com/webhook", cfg.WebhookURL)
	assert.Equal(t, "token-abc", cfg.WebhookSecretToken)
	assert.Equal(t, "8888", cfg.WebhookPort)
	assert.Equal(t, "/webhook", cfg.WebhookPath)
}

func TestConfigLoadWithWebhookURLWithPort(t *testing.T) {
	t.Setenv("WEBHOOK_URL", "https://13.140.43.93:8443/telegram/webhook")
	t.Setenv("WEBHOOK_PORT", "8082")

	cfg := config.Load()
	assert.Equal(t, "https://13.140.43.93:8443/telegram/webhook", cfg.WebhookURL)
	assert.Equal(t, "8443", cfg.WebhookPort)
	assert.Equal(t, "/telegram/webhook", cfg.WebhookPath)
}
