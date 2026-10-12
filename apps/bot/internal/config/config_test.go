package config_test

import (
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Isola variáveis de teste para garantir que valores de .env.local não sobrescrevam defaults
	t.Setenv("PORT", "")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("WEBHOOK_URL", "")
	t.Setenv("WEBHOOK_SECRET_TOKEN", "")
	t.Setenv("WEBHOOK_PORT", "")
	t.Setenv("WEBHOOK_PATH", "")

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
