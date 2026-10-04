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

func TestAPIClientSimulatePurchase(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/transactions/simulations", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req client.SimulationRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, int64(12345), req.TelegramID)
		assert.Equal(t, 2400.00, req.Amount)
		assert.Equal(t, 12, req.Installments)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"total_amount": 2400.00,
			"installments": 12,
			"installment_amount": 200.00,
			"current_cycle_s2s_before": 100.00,
			"current_cycle_s2s_after": 93.33,
			"current_cycle_reduction": 6.67,
			"current_cycle_reduction_percent": 6.67,
			"critical_cycle_number": 1,
			"critical_cycle_s2s": 93.33,
			"critical_cycle_health_status": "HEALTHY",
			"deficit_risk_alert": false,
			"recommendation_message": "Compra segura.",
			"cycles": [
				{
					"cycle_start": "2026-10-01",
					"cycle_end": "2026-10-31",
					"s2s_today": 93.33,
					"s2s_reduction": 6.67,
					"s2s_reduction_percent": 6.67,
					"projected_balance": 1800.00,
					"health_status": "HEALTHY",
					"bottleneck": false
				}
			]
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.SimulatePurchase(context.Background(), client.SimulationRequest{
		TelegramID:   12345,
		Amount:       2400.00,
		Installments: 12,
	})

	require.NoError(t, err)
	assert.Equal(t, 2400.00, resp.TotalAmount)
	assert.Equal(t, 12, resp.Installments)
	assert.Equal(t, 200.00, resp.InstallmentAmount)
	assert.Equal(t, 100.00, resp.CurrentCycleS2SBefore)
	assert.Equal(t, 93.33, resp.CurrentCycleS2SAfter)
	assert.Equal(t, "HEALTHY", resp.CriticalCycleHealthStatus)
	assert.False(t, resp.DeficitRiskAlert)
	assert.Len(t, resp.Cycles, 1)
}

func TestAPIClientDailyCheckIn(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/transactions/checkin", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		var req client.DailyCheckInRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, int64(12345), req.TelegramID)
		assert.Len(t, req.UntrackedExpenses, 1)
		assert.Equal(t, 35.00, req.UntrackedExpenses[0].Amount)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"s2s_calculated": 80.00,
			"spent_today": 35.00,
			"daily_quota": 80.00,
			"delta_savings": 45.00,
			"health_status": "HEALTHY",
			"next_day_s2s": 82.50,
			"days_remaining": 15,
			"message": "Parabéns! Você economizou R$ 45,00 hoje."
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.DailyCheckIn(context.Background(), client.DailyCheckInRequest{
		TelegramID: 12345,
		UntrackedExpenses: []client.CheckInExpenseInput{
			{Description: "Lanche", Amount: 35.00, CategoryName: "Alimentação"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 80.00, resp.S2SCalculated)
	assert.Equal(t, 35.00, resp.SpentToday)
	assert.Equal(t, 45.00, resp.DeltaSavings)
	assert.Equal(t, 82.50, resp.NextDayS2S)
	assert.Contains(t, resp.Message, "economizou")
}

func TestAPIClientAdjustAccountBalance(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/bank-accounts/balance", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		var req client.AdjustBalanceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, int64(12345), req.TelegramID)
		assert.Equal(t, 4800.00, req.NewBalance)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"account_id": "acc-123",
			"account_name": "Conta Corrente",
			"previous_balance": 5000.00,
			"new_balance": 4800.00,
			"total_liquid_balance": 4800.00,
			"message": "Saldo da conta Conta Corrente atualizado com sucesso."
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.AdjustAccountBalance(context.Background(), client.AdjustBalanceRequest{
		TelegramID: 12345,
		NewBalance: 4800.00,
	})

	require.NoError(t, err)
	assert.Equal(t, "acc-123", resp.AccountID)
	assert.Equal(t, 4800.00, resp.NewBalance)
	assert.Equal(t, 4800.00, resp.TotalLiquidBalance)
	assert.Contains(t, resp.Message, "atualizado com sucesso")
}

func TestAPIClientErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"telegram_id é obrigatório"}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, "secret")

	_, err := apiClient.SimulatePurchase(context.Background(), client.SimulationRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")

	_, err = apiClient.DailyCheckIn(context.Background(), client.DailyCheckInRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")

	_, err = apiClient.AdjustAccountBalance(context.Background(), client.AdjustBalanceRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")

	_, err = apiClient.RegisterIncome(context.Background(), client.IncomeRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestAPIClientRegisterIncome(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/transactions/income", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		var req client.IncomeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, int64(12345), req.TelegramID)
		assert.Equal(t, 5000.00, req.Amount)
		assert.Equal(t, "Salário", req.Description)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"transaction": {
				"id": "tx-inc-1",
				"description": "Salário",
				"amount": 5000.00
			},
			"category_name": "Renda",
			"previous_s2s": 50.00,
			"new_s2s": 216.66,
			"health_status": "HEALTHY",
			"days_remaining": 20,
			"total_liquid_balance": 5200.00
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.RegisterIncome(context.Background(), client.IncomeRequest{
		TelegramID:  12345,
		Amount:      5000.00,
		Description: "Salário",
	})

	require.NoError(t, err)
	assert.Equal(t, "tx-inc-1", resp.Transaction.ID)
	assert.Equal(t, "Renda", resp.CategoryName)
	assert.Equal(t, 50.00, resp.PreviousS2S)
	assert.Equal(t, 216.66, resp.NewS2S)
	assert.Equal(t, "HEALTHY", resp.HealthStatus)
	assert.Equal(t, 5200.00, resp.TotalLiquidBalance)
}

func TestAPIClient_ListInvestments_Success(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/investments", r.URL.Path)
		assert.Equal(t, "12345", r.URL.Query().Get("telegram_id"))
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"investments": [
				{
					"id": "inv-1",
					"ticker": "ALUP11",
					"quantity": 100,
					"average_price": 45.00,
					"total_cost": 4500.00
				},
				{
					"id": "inv-2",
					"ticker": "TD-SELIC",
					"quantity": "2.5",
					"average_price": 14000.00,
					"total_cost": 35000.00
				}
			],
			"total_invested": 39500.00,
			"total_liquid_balance": 5000.00,
			"total_net_worth": 44500.00
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.ListInvestments(context.Background(), 12345)

	require.NoError(t, err)
	assert.Len(t, resp.Investments, 2)
	assert.Equal(t, "ALUP11", resp.Investments[0].Ticker)
	assert.Equal(t, client.FlexFloat(100), resp.Investments[0].Quantity)
	assert.Equal(t, 45.00, resp.Investments[0].AveragePrice)
	assert.Equal(t, 4500.00, resp.Investments[0].TotalCost)
	assert.Equal(t, "TD-SELIC", resp.Investments[1].Ticker)
	assert.Equal(t, client.FlexFloat(2.5), resp.Investments[1].Quantity)
	assert.Equal(t, 39500.00, resp.TotalInvested)
	assert.Equal(t, 5000.00, resp.TotalLiquidBalance)
	assert.Equal(t, 44500.00, resp.TotalNetWorth)
}

func TestAPIClient_ListInvestments_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Usuário não encontrado"}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, "secret")
	resp, err := apiClient.ListInvestments(context.Background(), 99999)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "status 404")
	assert.Contains(t, err.Error(), "Usuário não encontrado")
}

func TestAPIClient_SellInvestment_Success(t *testing.T) {
	expectedSecret := "bot-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/investments/sell", r.URL.Path)
		assert.Equal(t, expectedSecret, r.Header.Get("X-Internal-Secret"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req client.SellInvestmentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, int64(12345), req.TelegramID)
		assert.Equal(t, "ALUP11", req.Ticker)
		assert.Equal(t, 50.0, req.Quantity)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ticker": "ALUP11",
			"sold_quantity": 50,
			"remaining_quantity": 150,
			"average_price": 45.00,
			"is_closed": false,
			"total_invested": 22750.00,
			"total_liquid_balance": 5000.00,
			"total_net_worth": 27750.00
		}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, expectedSecret)
	resp, err := apiClient.SellInvestment(context.Background(), client.SellInvestmentRequest{
		TelegramID: 12345,
		Ticker:     "ALUP11",
		Quantity:   50.0,
	})

	require.NoError(t, err)
	assert.Equal(t, "ALUP11", resp.Ticker)
	assert.Equal(t, client.FlexFloat(50), resp.SoldQuantity)
	assert.Equal(t, client.FlexFloat(150), resp.RemainingQuantity)
	assert.Equal(t, 45.00, resp.AveragePrice)
	assert.False(t, resp.IsClosed)
	assert.Equal(t, 22750.00, resp.TotalInvested)
	assert.Equal(t, 27750.00, resp.TotalNetWorth)
}

func TestAPIClient_SellInvestment_InsufficientQuantityError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"quantidade insuficiente para venda"}`))
	}))
	defer server.Close()

	apiClient := client.NewAPIClient(server.URL, "secret")
	resp, err := apiClient.SellInvestment(context.Background(), client.SellInvestmentRequest{
		TelegramID: 12345,
		Ticker:     "PETR4",
		Quantity:   100.0,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "status 400")
	assert.Contains(t, err.Error(), "quantidade insuficiente para venda")
}
