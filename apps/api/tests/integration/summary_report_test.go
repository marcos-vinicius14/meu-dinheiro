package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/botapi"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryReport_Unauthorized(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	req := httptest.NewRequest(http.MethodGet, "/internal/reports/summary?telegram_id=123456&period=today", nil)
	res := app.ExecuteRequest(req)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestSummaryReport_UserNotFound(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	req := httptest.NewRequest(http.MethodGet, "/internal/reports/summary?telegram_id=999999999&period=today", nil)
	req.Header.Set("X-Internal-Secret", app.InternalKey)
	res := app.ExecuteRequest(req)
	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestSummaryReport_DailySummary(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	telegramID := int64(77889901)

	// 1. Onboarding do usuário
	onboardingBody, _ := json.Marshal(map[string]any{
		"telegram_id":     telegramID,
		"first_name":      "Marcos Resumo",
		"initial_balance": 3000.00,
		"cycle_start_day": 1,
		"fixed_expenses":  []map[string]any{},
	})
	onbReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(onboardingBody))
	onbReq.Header.Set("X-Internal-Secret", app.InternalKey)
	onbRes := app.ExecuteRequest(onbReq)
	require.Equal(t, http.StatusOK, onbRes.Code)

	// 2. Lança 2 despesas no dia de teste (2026-10-12) e 1 em outro dia (2026-10-10)
	testDate := "2026-10-12"

	exp1, _ := json.Marshal(map[string]any{
		"telegram_id":   telegramID,
		"amount":        35.50,
		"description":   "Almoço Executivo",
		"category_name": "Alimentação",
		"date":          testDate,
	})
	req1 := httptest.NewRequest(http.MethodPost, "/internal/transactions/quick-expense", bytes.NewReader(exp1))
	req1.Header.Set("X-Internal-Secret", app.InternalKey)
	res1 := app.ExecuteRequest(req1)
	require.Equal(t, http.StatusCreated, res1.Code)

	exp2, _ := json.Marshal(map[string]any{
		"telegram_id":   telegramID,
		"amount":        15.00,
		"description":   "Café e Pão de Queijo",
		"category_name": "Alimentação",
		"date":          testDate,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/internal/transactions/quick-expense", bytes.NewReader(exp2))
	req2.Header.Set("X-Internal-Secret", app.InternalKey)
	res2 := app.ExecuteRequest(req2)
	require.Equal(t, http.StatusCreated, res2.Code)

	expOtherDay, _ := json.Marshal(map[string]any{
		"telegram_id":   telegramID,
		"amount":        100.00,
		"description":   "Gasto de outro dia",
		"category_name": "Transporte",
		"date":          "2026-10-10",
	})
	reqOther := httptest.NewRequest(http.MethodPost, "/internal/transactions/quick-expense", bytes.NewReader(expOtherDay))
	reqOther.Header.Set("X-Internal-Secret", app.InternalKey)
	resOther := app.ExecuteRequest(reqOther)
	require.Equal(t, http.StatusCreated, resOther.Code)

	// 3. Consulta resumo diário
	url := fmt.Sprintf("/internal/reports/summary?telegram_id=%d&period=today&date=%s", telegramID, testDate)
	sumReq := httptest.NewRequest(http.MethodGet, url, nil)
	sumReq.Header.Set("X-Internal-Secret", app.InternalKey)
	sumRes := app.ExecuteRequest(sumReq)
	require.Equal(t, http.StatusOK, sumRes.Code)

	var resp botapi.SummaryReportResponse
	err := json.NewDecoder(sumRes.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "today", resp.Period)
	assert.True(t, resp.AccountBalance.IsPositive())
	assert.True(t, resp.TotalLiquidBalance.IsPositive())
	assert.NotEmpty(t, resp.HealthStatus)
	assert.True(t, resp.S2SToday.IsPositive())

	require.NotNil(t, resp.Today)
	assert.Equal(t, testDate, resp.Today.Date)
	assert.Equal(t, "50.50", resp.Today.TotalSpent.String()) // 35.50 + 15.00
	assert.Equal(t, 2, resp.Today.TransactionsCount)
	require.Len(t, resp.Today.Transactions, 2)
	assert.Equal(t, "Café e Pão de Queijo", resp.Today.Transactions[0].Description)
	assert.Equal(t, "Almoço Executivo", resp.Today.Transactions[1].Description)
}

func TestSummaryReport_WeeklySummary(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	telegramID := int64(77889902)

	// 1. Onboarding
	onboardingBody, _ := json.Marshal(map[string]any{
		"telegram_id":     telegramID,
		"first_name":      "Semanal User",
		"initial_balance": 5000.00,
		"cycle_start_day": 1,
		"fixed_expenses":  []map[string]any{},
	})
	onbReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(onboardingBody))
	onbReq.Header.Set("X-Internal-Secret", app.InternalKey)
	onbRes := app.ExecuteRequest(onbReq)
	require.Equal(t, http.StatusOK, onbRes.Code)

	// Semana de 12/10/2026 (Segunda) até 18/10/2026 (Domingo).
	// Lança despesas em 12/10 (Segunda), 13/10 (Terça) e 14/10 (Quarta).
	expenses := []struct {
		desc   string
		amount float64
		cat    string
		date   string
	}{
		{"Mercado Semanal", 200.00, "Alimentação", "2026-10-12"},
		{"Almoço", 50.00, "Alimentação", "2026-10-13"},
		{"Combustível", 100.00, "Transporte", "2026-10-13"},
		{"Cinema", 50.00, "Lazer", "2026-10-14"},
		{"Gasto Semana Passada", 300.00, "Alimentação", "2026-10-05"}, // fora da semana
	}

	for _, e := range expenses {
		b, _ := json.Marshal(map[string]any{
			"telegram_id":   telegramID,
			"amount":        e.amount,
			"description":   e.desc,
			"category_name": e.cat,
			"date":          e.date,
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/transactions/quick-expense", bytes.NewReader(b))
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		require.Equal(t, http.StatusCreated, res.Code)
	}

	// 2. Consulta resumo semanal com data base 14/10/2026 (Quarta-feira -> 3 dias decorridos)
	url := fmt.Sprintf("/internal/reports/summary?telegram_id=%d&period=week&date=2026-10-14", telegramID)
	sumReq := httptest.NewRequest(http.MethodGet, url, nil)
	sumReq.Header.Set("X-Internal-Secret", app.InternalKey)
	sumRes := app.ExecuteRequest(sumReq)
	require.Equal(t, http.StatusOK, sumRes.Code)

	var resp botapi.SummaryReportResponse
	err := json.NewDecoder(sumRes.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "week", resp.Period)
	require.NotNil(t, resp.Week)

	assert.Equal(t, "2026-10-12", resp.Week.StartDate)
	assert.Equal(t, "2026-10-18", resp.Week.EndDate)
	assert.Equal(t, 3, resp.Week.DaysElapsed)

	// Total na semana: 200 + 50 + 100 + 50 = 400.00
	assert.Equal(t, "400.00", resp.Week.TotalSpent.String())
	assert.Equal(t, 4, resp.Week.TransactionsCount)

	// Média diária: 400.00 / 3 = 133.33
	assert.Equal(t, "133.33", resp.Week.DailyAverage.String())

	// Top Categorias:
	// 1. Alimentação: 250.00 (62.50%)
	// 2. Transporte: 100.00 (25.00%)
	// 3. Lazer: 50.00 (12.50%)
	require.Len(t, resp.Week.TopCategories, 3)
	assert.Equal(t, "Alimentação", resp.Week.TopCategories[0].CategoryName)
	assert.Equal(t, "250.00", resp.Week.TopCategories[0].Amount.String())
	assert.Equal(t, "62.50", resp.Week.TopCategories[0].Percentage.StringFixed(2))

	assert.Equal(t, "Transporte", resp.Week.TopCategories[1].CategoryName)
	assert.Equal(t, "100.00", resp.Week.TopCategories[1].Amount.String())
	assert.Equal(t, "25.00", resp.Week.TopCategories[1].Percentage.StringFixed(2))

	assert.Equal(t, "Lazer", resp.Week.TopCategories[2].CategoryName)
	assert.Equal(t, "50.00", resp.Week.TopCategories[2].Amount.String())
	assert.Equal(t, "12.50", resp.Week.TopCategories[2].Percentage.StringFixed(2))
}
