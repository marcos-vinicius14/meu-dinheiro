package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalBotAPIFullFlow(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	telegramID := int64(88990011)

	// 1. Testa bloqueio de acesso sem cabeçalho de chave interna (403 Forbidden)
	unauthReq := httptest.NewRequest(http.MethodGet, "/internal/users/context-by-telegram?telegram_id=88990011", nil)
	unauthRes := app.ExecuteRequest(unauthReq)
	assert.Equal(t, http.StatusForbidden, unauthRes.Code)

	// 2. Executa Onboarding via Bot com Gastos Essenciais, Escolha de Reserva 6x e Ciclo no dia 05
	onboardingBody, _ := json.Marshal(map[string]any{
		"telegram_id":     telegramID,
		"first_name":      "Julius Bot User",
		"username":        "julius_user",
		"initial_balance": 5000.00,
		"cycle_start_day": 5,
		"fixed_expenses": []map[string]any{
			{"description": "Aluguel", "amount": 1500.00, "category_name": "Moradia"},
			{"description": "Supermercado", "amount": 800.00, "category_name": "Alimentação"},
			{"description": "Farmácia", "amount": 200.00, "category_name": "Saúde"},
		},
		"emergency_fund_months": 6,
		"target_savings":        500.00,
		"flexible_budget_cap":   2000.00,
		"investments": []map[string]any{
			{"ticker": "ALUP11", "quantity": 10, "average_price": 42.23},
		},
	})

	onboardingReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(onboardingBody))
	onboardingReq.Header.Set("X-Internal-Secret", app.InternalKey)
	onboardingRes := app.ExecuteRequest(onboardingReq)
	require.Equal(t, http.StatusOK, onboardingRes.Code)

	var onbResp struct {
		Message string    `json:"message"`
		UserID  uuid.UUID `json:"user_id"`
		Cycle   struct {
			StartDate     string  `json:"start_date"`
			EndDate       string  `json:"end_date"`
			DaysRemaining int     `json:"days_remaining"`
			S2SToday      float64 `json:"s2s_today"`
			HealthStatus  string  `json:"health_status"`
		} `json:"cycle"`
		EmergencyFund struct {
			MonthlyEssentialCost float64 `json:"monthly_essential_cost"`
			Suggested6x          float64 `json:"suggested_6x"`
			Suggested12x         float64 `json:"suggested_12x"`
			ChosenTarget         float64 `json:"chosen_target"`
			ChosenMonths         int     `json:"chosen_months"`
			MonthsCovered        float64 `json:"months_covered"`
			ProgressPercent      string  `json:"progress_percent"`
		} `json:"emergency_fund"`
		TotalLiquidBalance float64 `json:"total_liquid_balance"`
		TotalInvested      float64 `json:"total_invested"`
		TotalNetWorth      float64 `json:"total_net_worth"`
	}

	err := json.NewDecoder(onboardingRes.Body).Decode(&onbResp)
	require.NoError(t, err)
	assert.Equal(t, "Onboarding concluído com sucesso", onbResp.Message)
	assert.Equal(t, 2500.00, onbResp.EmergencyFund.MonthlyEssentialCost) // 1500 + 800 + 200
	assert.Equal(t, 15000.00, onbResp.EmergencyFund.Suggested6x)         // 2500 * 6
	assert.Equal(t, 30000.00, onbResp.EmergencyFund.Suggested12x)        // 2500 * 12
	assert.Equal(t, 15000.00, onbResp.EmergencyFund.ChosenTarget)
	assert.Equal(t, 6, onbResp.EmergencyFund.ChosenMonths)
	assert.Equal(t, 2.0, onbResp.EmergencyFund.MonthsCovered)       // 5000 / 2500
	assert.Equal(t, "33.33", onbResp.EmergencyFund.ProgressPercent) // (5000 / 15000) * 100
	assert.Equal(t, 5000.00, onbResp.TotalLiquidBalance)
	assert.Equal(t, 422.30, onbResp.TotalInvested)
	assert.Equal(t, 5422.30, onbResp.TotalNetWorth)
	assert.True(t, onbResp.Cycle.S2SToday > 0)
	assert.True(t, onbResp.Cycle.DaysRemaining > 0)

	// 3. Consulta o contexto financeiro completo do usuário via Telegram ID
	contextReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/users/context-by-telegram?telegram_id=%d", telegramID), nil)
	contextReq.Header.Set("X-Internal-Secret", app.InternalKey)
	contextRes := app.ExecuteRequest(contextReq)
	require.Equal(t, http.StatusOK, contextRes.Code)

	var ctxResp struct {
		User struct {
			TelegramID    int64   `json:"telegram_id"`
			CycleStartDay int     `json:"cycle_start_day"`
			TargetSavings float64 `json:"target_savings"`
		} `json:"user"`
		Cycle struct {
			S2SToday     float64 `json:"s2s_today"`
			HealthStatus string  `json:"health_status"`
		} `json:"cycle"`
		EmergencyFund struct {
			MonthlyEssentialCost float64 `json:"monthly_essential_cost"`
			Target               float64 `json:"target"`
		} `json:"emergency_fund"`
		TotalLiquidBalance float64 `json:"total_liquid_balance"`
		TotalInvested      float64 `json:"total_invested"`
		TotalNetWorth      float64 `json:"total_net_worth"`
	}
	err = json.NewDecoder(contextRes.Body).Decode(&ctxResp)
	require.NoError(t, err)
	assert.Equal(t, telegramID, ctxResp.User.TelegramID)
	assert.Equal(t, 5, ctxResp.User.CycleStartDay)
	assert.Equal(t, 500.00, ctxResp.User.TargetSavings)
	assert.Equal(t, 2500.00, ctxResp.EmergencyFund.MonthlyEssentialCost)
	assert.Equal(t, 15000.00, ctxResp.EmergencyFund.Target)
	assert.Equal(t, 5422.30, ctxResp.TotalNetWorth)

	// 4. Lança uma despesa rápida via Bot (/gasto 50.00 Lanche)
	initialS2S := ctxResp.Cycle.S2SToday
	expenseBody, _ := json.Marshal(map[string]any{
		"telegram_id":   telegramID,
		"amount":        50.00,
		"description":   "Lanche",
		"category_name": "Alimentação",
	})
	expenseReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/quick-expense", bytes.NewReader(expenseBody))
	expenseReq.Header.Set("X-Internal-Secret", app.InternalKey)
	expenseRes := app.ExecuteRequest(expenseReq)
	require.Equal(t, http.StatusCreated, expenseRes.Code)

	var expResp struct {
		CategoryName string  `json:"category_name"`
		PreviousS2S  float64 `json:"previous_s2s"`
		NewS2S       float64 `json:"new_s2s"`
		HealthStatus string  `json:"health_status"`
	}
	err = json.NewDecoder(expenseRes.Body).Decode(&expResp)
	require.NoError(t, err)
	assert.Equal(t, "Alimentação", expResp.CategoryName)
	assert.Equal(t, initialS2S, expResp.PreviousS2S)
	assert.True(t, expResp.NewS2S < expResp.PreviousS2S, "Novo S2S deve ser menor após registrar o gasto")

	// 5. Adiciona novo investimento via Bot (/investimento BBAS3 20 30.00)
	investBody, _ := json.Marshal(map[string]any{
		"telegram_id": telegramID,
		"ticker":      "BBAS3",
		"quantity":    20,
		"price":       30.00,
	})
	investReq := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(investBody))
	investReq.Header.Set("X-Internal-Secret", app.InternalKey)
	investRes := app.ExecuteRequest(investReq)
	require.Equal(t, http.StatusOK, investRes.Code)

	var addInvResp struct {
		Investment struct {
			Ticker       string  `json:"ticker"`
			Quantity     string  `json:"quantity"`
			AveragePrice float64 `json:"average_price"`
			TotalCost    float64 `json:"total_cost"`
		} `json:"investment"`
		TotalInvested float64 `json:"total_invested"`
		TotalNetWorth float64 `json:"total_net_worth"`
	}
	err = json.NewDecoder(investRes.Body).Decode(&addInvResp)
	require.NoError(t, err)
	assert.Equal(t, "BBAS3", addInvResp.Investment.Ticker)
	assert.Equal(t, "20", addInvResp.Investment.Quantity)
	assert.Equal(t, 30.00, addInvResp.Investment.AveragePrice)
	assert.Equal(t, 600.00, addInvResp.Investment.TotalCost)
	assert.Equal(t, 1022.30, addInvResp.TotalInvested) // 422.30 (ALUP11) + 600 (BBAS3)
}

func TestInternalBotAPI_OnboardingValidationErrors(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	// 1. Corpo inválido (JSON quebrado)
	invalidJSONReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader([]byte("{invalid-json")))
	invalidJSONReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res := app.ExecuteRequest(invalidJSONReq)
	assert.Equal(t, http.StatusBadRequest, res.Code)

	// 2. Falta telegram_id e first_name
	missingFieldsBody, _ := json.Marshal(map[string]any{
		"initial_balance": 1000.00,
	})
	missingFieldsReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(missingFieldsBody))
	missingFieldsReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res = app.ExecuteRequest(missingFieldsReq)
	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "telegram_id e first_name são obrigatórios")

	// 3. Fallback defensivo de cycle_start_day > 28 e emergency_fund_months inválido
	defensiveBody, _ := json.Marshal(map[string]any{
		"telegram_id":           int64(778899),
		"first_name":            "Defensive User",
		"initial_balance":       3000.00,
		"cycle_start_day":       35, // Fora do intervalo 1..28 -> deve cair para 1
		"emergency_fund_months": 24, // Diferente de 6 e 12 -> deve cair para 6
		"target_savings":        200.00,
	})
	defensiveReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(defensiveBody))
	defensiveReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res = app.ExecuteRequest(defensiveReq)
	require.Equal(t, http.StatusOK, res.Code)

	var onbResp struct {
		EmergencyFund struct {
			ChosenMonths int `json:"chosen_months"`
		} `json:"emergency_fund"`
	}
	err := json.NewDecoder(res.Body).Decode(&onbResp)
	require.NoError(t, err)
	assert.Equal(t, 6, onbResp.EmergencyFund.ChosenMonths, "mês da reserva deve ter fallback para 6")

	// Verifica se cycle_start_day foi salvo como 1 no banco
	ctxReq := httptest.NewRequest(http.MethodGet, "/internal/users/context-by-telegram?telegram_id=778899", nil)
	ctxReq.Header.Set("X-Internal-Secret", app.InternalKey)
	ctxRes := app.ExecuteRequest(ctxReq)
	require.Equal(t, http.StatusOK, ctxRes.Code)

	var ctxResp struct {
		User struct {
			CycleStartDay int `json:"cycle_start_day"`
		} `json:"user"`
	}
	err = json.NewDecoder(ctxRes.Body).Decode(&ctxResp)
	require.NoError(t, err)
	assert.Equal(t, 1, ctxResp.User.CycleStartDay, "cycle_start_day deve ter fallback defensivo para 1")
}

func TestInternalBotAPI_ContextByTelegram_NotFound(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	// 1. telegram_id ausente
	emptyReq := httptest.NewRequest(http.MethodGet, "/internal/users/context-by-telegram", nil)
	emptyReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res := app.ExecuteRequest(emptyReq)
	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "telegram_id é obrigatório")

	// 2. telegram_id inválido (não numérico)
	invalidReq := httptest.NewRequest(http.MethodGet, "/internal/users/context-by-telegram?telegram_id=abc", nil)
	invalidReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res = app.ExecuteRequest(invalidReq)
	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "telegram_id inválido")

	// 3. telegram_id inexistente
	notFoundReq := httptest.NewRequest(http.MethodGet, "/internal/users/context-by-telegram?telegram_id=9999999999", nil)
	notFoundReq.Header.Set("X-Internal-Secret", app.InternalKey)
	res = app.ExecuteRequest(notFoundReq)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Contains(t, res.Body.String(), "Usuário não encontrado")
}
