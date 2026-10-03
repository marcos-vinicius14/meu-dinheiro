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

func TestAPIClientSaveOnboarding(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/users/onboarding", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"message": "Onboarding concluído com sucesso",
			"user_id": "11111111-1111-1111-1111-111111111111",
			"cycle": { "start_date": "2026-09-01", "end_date": "2026-09-30", "days_remaining": 30, "s2s_today": 83.33, "health_status": "HEALTHY" },
			"emergency_fund": { "monthly_essential_cost": 2500.00, "suggested_6x": 15000.00, "suggested_12x": 30000.00, "chosen_target": 15000.00, "chosen_months": 6, "months_covered": 2.0, "progress_percent": 33.33 },
			"total_liquid_balance": 5000.00,
			"total_invested": 0.00,
			"total_net_worth": 5000.00
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.SaveOnboarding(context.Background(), client.OnboardingRequest{
		TelegramID:          12345,
		FirstName:           "Marcos",
		InitialBalance:      5000.00,
		CycleStartDay:       1,
		EmergencyFundMonths: 6,
		TargetSavings:       500.00,
	})

	require.NoError(t, err)
	assert.Equal(t, "Onboarding concluído com sucesso", resp.Message)
	assert.Equal(t, 83.33, resp.Cycle.S2SToday)
	assert.Equal(t, 15000.00, resp.EmergencyFund.ChosenTarget)
}

func TestAPIClientGetUserContextByTelegram(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/users/context-by-telegram", r.URL.Path)
		assert.Equal(t, "12345", r.URL.Query().Get("telegram_id"))
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"user": { "id": "111", "telegram_id": 12345, "first_name": "Marcos", "cycle_start_day": 1 },
			"total_liquid_balance": 3500.00,
			"cycle": { "s2s_today": 75.00, "health_status": "HEALTHY", "days_remaining": 15 },
			"emergency_fund": { "monthly_essential_cost": 2000.00, "target": 12000.00, "months": 6, "months_covered": 1.75, "progress_percent": 29.16 },
			"investments": [],
			"total_invested": 0.00,
			"total_net_worth": 3500.00
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	ctx, err := apiClient.GetUserContextByTelegram(context.Background(), 12345)
	require.NoError(t, err)
	assert.Equal(t, 12345, int(ctx.User.TelegramID))
	assert.Equal(t, 75.00, ctx.Cycle.S2SToday)
	assert.Equal(t, 3500.00, ctx.TotalLiquidBalance)
}

func TestAPIClientAddInvestment(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/investments", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"investment": { "id": "inv-1", "ticker": "ALUP11", "quantity": 10.0, "average_price": 42.23, "total_cost": 422.30 },
			"total_invested": 422.30,
			"total_net_worth": 3922.30
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.AddInvestment(context.Background(), client.AddInvestmentRequest{
		TelegramID: 12345,
		Ticker:     "ALUP11",
		Quantity:   10,
		Price:      42.23,
	})
	require.NoError(t, err)
	assert.Equal(t, "ALUP11", resp.Investment.Ticker)
	assert.Equal(t, 422.30, resp.TotalInvested)
}

func TestAPIClientQuickExpense(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/transactions/quick-expense", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"transaction": { "id": "tx-1", "description": "Almoço", "amount": 34.90 },
			"category_name": "Alimentação",
			"previous_s2s": 85.00,
			"new_s2s": 82.50,
			"health_status": "HEALTHY",
			"days_remaining": 14
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.QuickExpense(context.Background(), client.QuickExpenseRequest{
		TelegramID:  12345,
		Amount:      34.90,
		Description: "Almoço",
	})
	require.NoError(t, err)
	assert.Equal(t, "Alimentação", resp.CategoryName)
	assert.Equal(t, 82.50, resp.NewS2S)
	assert.Equal(t, 85.00, resp.PreviousS2S)
}
