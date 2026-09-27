package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClientAuthorizeChallenge(t *testing.T) {
	expectedSecret := "secret-key"
	expectedToken := "test-challenge-token"
	expectedTelegramID := int64(123456789)
	expectedName := "John Doe"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auth/authorize-challenge", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body client.AuthorizeChallengeRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, expectedToken, body.Token)
		assert.Equal(t, expectedTelegramID, body.TelegramID)
		assert.Equal(t, expectedName, body.Name)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"authorized"}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	err := apiClient.AuthorizeChallenge(context.Background(), expectedToken, expectedTelegramID, expectedName)
	assert.NoError(t, err)
}

func TestAPIClientAuthorizeChallenge_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"challenge nao encontrado"}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, "secret")
	err := apiClient.AuthorizeChallenge(context.Background(), "invalid-token", 123, "Name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}
