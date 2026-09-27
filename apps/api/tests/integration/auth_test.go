package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramAuthFlow(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	// 1. Web solicita criação de challenge
	createReq := httptest.NewRequest(http.MethodPost, "/auth/telegram/challenge", nil)
	createRes := app.ExecuteRequest(createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	var challenge struct {
		Token  string `json:"token"`
		BotURL string `json:"bot_url"`
	}
	err := json.NewDecoder(createRes.Body).Decode(&challenge)
	require.NoError(t, err)
	assert.NotEmpty(t, challenge.Token)
	assert.Contains(t, challenge.BotURL, "t.me/MeuDinheiroTestBot?start=auth_"+challenge.Token)

	// 2. Antes do bot autorizar, o polling deve retornar status PENDING
	pollReq := httptest.NewRequest(http.MethodGet, "/auth/telegram/poll?token="+challenge.Token, nil)
	pollRes := app.ExecuteRequest(pollReq)
	require.Equal(t, http.StatusOK, pollRes.Code)

	var pendingStatus struct {
		Status string `json:"status"`
	}
	err = json.NewDecoder(pollRes.Body).Decode(&pendingStatus)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", pendingStatus.Status)

	// 3. Bot autoriza o challenge
	authBody, _ := json.Marshal(map[string]any{
		"token":       challenge.Token,
		"telegram_id": 99887766,
		"username":    "marcos_tele",
		"first_name":  "Marcos",
	})
	authReq := httptest.NewRequest(http.MethodPost, "/internal/auth/authorize-challenge", bytes.NewReader(authBody))
	authReq.Header.Set("X-Internal-API-Key", "test-internal-api-key")
	authReq.Header.Set("Content-Type", "application/json")
	authRes := app.ExecuteRequest(authReq)
	require.Equal(t, http.StatusOK, authRes.Code)

	// 4. Web refaz polling e agora recebe status AUTHORIZED e o cookie de sessão
	pollReq2 := httptest.NewRequest(http.MethodGet, "/auth/telegram/poll?token="+challenge.Token, nil)
	pollRes2 := app.ExecuteRequest(pollReq2)
	require.Equal(t, http.StatusOK, pollRes2.Code)

	var authorizedStatus struct {
		Status string `json:"status"`
		Token  string `json:"token"`
		User   struct {
			TelegramID int64  `json:"telegram_id"`
			FirstName  string `json:"first_name"`
		} `json:"user"`
	}
	err = json.NewDecoder(pollRes2.Body).Decode(&authorizedStatus)
	require.NoError(t, err)
	assert.Equal(t, "AUTHORIZED", authorizedStatus.Status)
	assert.Equal(t, int64(99887766), authorizedStatus.User.TelegramID)
	assert.Equal(t, "Marcos", authorizedStatus.User.FirstName)
	assert.NotEmpty(t, authorizedStatus.Token)

	// Verifica se o cookie session_token foi enviado no Set-Cookie
	cookies := pollRes2.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, authorizedStatus.Token, sessionCookie.Value)
	assert.True(t, sessionCookie.HttpOnly)

	// 5. Acessa endpoint protegido /auth/me usando o cookie de sessão
	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(sessionCookie)
	meRes := app.ExecuteRequest(meReq)
	require.Equal(t, http.StatusOK, meRes.Code)

	var me struct {
		TelegramID int64  `json:"telegram_id"`
		FirstName  string `json:"first_name"`
	}
	err = json.NewDecoder(meRes.Body).Decode(&me)
	require.NoError(t, err)
	assert.Equal(t, int64(99887766), me.TelegramID)

	// 6. Faz logout e verifica cookie expirado
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRes := app.ExecuteRequest(logoutReq)
	assert.Equal(t, http.StatusNoContent, logoutRes.Code)

	var expiredCookie *http.Cookie
	for _, c := range logoutRes.Result().Cookies() {
		if c.Name == "session_token" {
			expiredCookie = c
			break
		}
	}
	require.NotNil(t, expiredCookie)
	assert.Equal(t, -1, expiredCookie.MaxAge)
}
