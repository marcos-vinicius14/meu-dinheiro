package bot_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestBot(cfg *config.Config) *bot.Bot {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiClient := client.NewAPIClient("http://localhost:8080", "test-key")
	return bot.NewBot(nil, apiClient, logger, cfg)
}

func TestWebhookHandler_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		WebhookPath: "/webhook",
		WebhookPort: "8081",
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"UP"`)
	assert.Contains(t, rr.Body.String(), `"mode":"webhook"`)
}

func TestWebhookHandler_SecretToken_Unauthorized(t *testing.T) {
	cfg := &config.Config{
		WebhookPath:        "/webhook",
		WebhookSecretToken: "correct-secret-token-123",
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	reqNoHeader := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(`{"update_id":1}`))
	rrNoHeader := httptest.NewRecorder()
	handler.ServeHTTP(rrNoHeader, reqNoHeader)
	assert.Equal(t, http.StatusUnauthorized, rrNoHeader.Code)

	reqWrong := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(`{"update_id":1}`))
	reqWrong.Header.Set(bot.HeaderTelegramSecretToken, "wrong-token")
	rrWrong := httptest.NewRecorder()
	handler.ServeHTTP(rrWrong, reqWrong)
	assert.Equal(t, http.StatusUnauthorized, rrWrong.Code)
}

func TestWebhookHandler_SecretToken_Success(t *testing.T) {
	cfg := &config.Config{
		WebhookPath:        "/webhook",
		WebhookSecretToken: "correct-secret-token-123",
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	validBody := `{
		"update_id": 1001,
		"message": {
			"message_id": 1,
			"date": 1727440000,
			"chat": {"id": 12345, "type": "private"},
			"from": {"id": 12345, "is_bot": false, "first_name": "Test"},
			"text": "/ajuda"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(validBody))
	req.Header.Set(bot.HeaderTelegramSecretToken, "correct-secret-token-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		WebhookPath: "/webhook",
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(`{invalid_json`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		WebhookPath: "/webhook",
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestWebhookHandler_NoSecretTokenConfigured(t *testing.T) {
	cfg := &config.Config{
		WebhookPath:        "/webhook",
		WebhookSecretToken: "", // Sem verificação de secret token
	}
	b := setupTestBot(cfg)
	handler := b.NewWebhookHandler()

	validBody := `{"update_id": 2002}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(validBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
