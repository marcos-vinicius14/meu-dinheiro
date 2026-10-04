package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalBotAPI_Investments(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	telegramID := int64(99112233)

	// Setup do usuário via Onboarding com R$ 5.000,00 de saldo inicial
	onboardingBody, _ := json.Marshal(map[string]any{
		"telegram_id":           telegramID,
		"first_name":            "Investidor Teste",
		"username":              "invest_tester",
		"initial_balance":       5000.00,
		"cycle_start_day":       5,
		"fixed_expenses":        []map[string]any{},
		"emergency_fund_months": 6,
		"target_savings":        500.00,
		"flexible_budget_cap":   3000.00,
		"investments":           []map[string]any{},
	})

	onbReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(onboardingBody))
	onbReq.Header.Set("X-Internal-Secret", app.InternalKey)
	onbRes := app.ExecuteRequest(onbReq)
	require.Equal(t, http.StatusOK, onbRes.Code)

	// Cenário 1: Segurança e Acesso Restrito (T-03-01)
	t.Run("Segurança: acesso restrito a chave interna", func(t *testing.T) {
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/investments?telegram_id=%d", telegramID), nil)
		resGet := app.ExecuteRequest(reqGet)
		assert.Equal(t, http.StatusForbidden, resGet.Code)

		sellBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    10,
		})
		reqSell := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(sellBody))
		resSell := app.ExecuteRequest(reqSell)
		assert.Equal(t, http.StatusForbidden, resSell.Code)
	})

	// Cenário 2: Consulta de Carteira Vazia (GET /internal/investments)
	t.Run("Consulta de Carteira Vazia", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/investments?telegram_id=%d", telegramID), nil)
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		require.Equal(t, http.StatusOK, res.Code)

		var resp struct {
			Investments        []any      `json:"investments"`
			TotalInvested      moneyFloat `json:"total_invested"`
			TotalLiquidBalance moneyFloat `json:"total_liquid_balance"`
			TotalNetWorth      moneyFloat `json:"total_net_worth"`
		}
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)

		assert.Empty(t, resp.Investments)
		assert.Equal(t, 0.00, float64(resp.TotalInvested))
		assert.Equal(t, 5000.00, float64(resp.TotalLiquidBalance))
		assert.Equal(t, 5000.00, float64(resp.TotalNetWorth))
	})

	// Cenário 3: Aporte de Múltiplos Ativos e Consulta Povoada
	t.Run("Aporte de Múltiplos Ativos e Consulta Povoada", func(t *testing.T) {
		// Aporte 1: ALUP11 (100 un a R$ 40,00)
		body1, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    100,
			"price":       40.00,
		})
		req1 := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(body1))
		req1.Header.Set("X-Internal-Secret", app.InternalKey)
		res1 := app.ExecuteRequest(req1)
		require.Equal(t, http.StatusOK, res1.Code)

		// Aporte 2: ALUP11 (100 un a R$ 50,00) -> PM recalculado para R$ 45,00 (Total R$ 9.000,00)
		body2, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    100,
			"price":       50.00,
		})
		req2 := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(body2))
		req2.Header.Set("X-Internal-Secret", app.InternalKey)
		res2 := app.ExecuteRequest(req2)
		require.Equal(t, http.StatusOK, res2.Code)

		// Ativo 2: PETR4 (50 un a R$ 30,00) -> Total R$ 1.500,00
		body3, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "PETR4",
			"quantity":    50,
			"price":       30.00,
		})
		req3 := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(body3))
		req3.Header.Set("X-Internal-Secret", app.InternalKey)
		res3 := app.ExecuteRequest(req3)
		require.Equal(t, http.StatusOK, res3.Code)

		// Ativo 3: TD-SELIC (1 un a R$ 14.500,00) -> Total R$ 14.500,00
		body4, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "TD-SELIC",
			"quantity":    1,
			"price":       14500.00,
		})
		req4 := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(body4))
		req4.Header.Set("X-Internal-Secret", app.InternalKey)
		res4 := app.ExecuteRequest(req4)
		require.Equal(t, http.StatusOK, res4.Code)

		// Consulta GET /internal/investments
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/investments?telegram_id=%d", telegramID), nil)
		reqGet.Header.Set("X-Internal-Secret", app.InternalKey)
		resGet := app.ExecuteRequest(reqGet)
		require.Equal(t, http.StatusOK, resGet.Code)

		var listResp struct {
			Investments []struct {
				Ticker       string     `json:"ticker"`
				Quantity     moneyFloat `json:"quantity"`
				AveragePrice moneyFloat `json:"average_price"`
				TotalCost    moneyFloat `json:"total_cost"`
			} `json:"investments"`
			TotalInvested      moneyFloat `json:"total_invested"`
			TotalLiquidBalance moneyFloat `json:"total_liquid_balance"`
			TotalNetWorth      moneyFloat `json:"total_net_worth"`
		}
		err := json.NewDecoder(resGet.Body).Decode(&listResp)
		require.NoError(t, err)

		assert.Len(t, listResp.Investments, 3)
		assert.Equal(t, 25000.00, float64(listResp.TotalInvested))     // 9000 + 1500 + 14500
		assert.Equal(t, 5000.00, float64(listResp.TotalLiquidBalance)) // 5000
		assert.Equal(t, 30000.00, float64(listResp.TotalNetWorth))     // 25000 + 5000
	})

	// Cenário 4: Venda Parcial (POST /internal/investments/sell)
	t.Run("Venda Parcial com PM inalterado", func(t *testing.T) {
		sellBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    50,
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(sellBody))
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		require.Equal(t, http.StatusOK, res.Code)

		var sellResp struct {
			Ticker             string     `json:"ticker"`
			SoldQuantity       moneyFloat `json:"sold_quantity"`
			RemainingQuantity  moneyFloat `json:"remaining_quantity"`
			AveragePrice       moneyFloat `json:"average_price"`
			IsClosed           bool       `json:"is_closed"`
			TotalInvested      moneyFloat `json:"total_invested"`
			TotalLiquidBalance moneyFloat `json:"total_liquid_balance"`
			TotalNetWorth      moneyFloat `json:"total_net_worth"`
		}
		err := json.NewDecoder(res.Body).Decode(&sellResp)
		require.NoError(t, err)

		assert.Equal(t, "ALUP11", sellResp.Ticker)
		assert.Equal(t, 50.00, float64(sellResp.SoldQuantity))
		assert.Equal(t, 150.00, float64(sellResp.RemainingQuantity))
		assert.Equal(t, 45.00, float64(sellResp.AveragePrice))
		assert.False(t, sellResp.IsClosed)
		assert.Equal(t, 22750.00, float64(sellResp.TotalInvested)) // 150*45 (6750) + 1500 + 14500 = 22750
		assert.Equal(t, 27750.00, float64(sellResp.TotalNetWorth)) // 22750 + 5000
	})

	// Cenário 5: Venda de 100% e Encerramento de Posição (D-09, D-12)
	t.Run("Venda de 100% e Encerramento de Posição", func(t *testing.T) {
		sellBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    150,
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(sellBody))
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		require.Equal(t, http.StatusOK, res.Code)

		var sellResp struct {
			RemainingQuantity moneyFloat `json:"remaining_quantity"`
			IsClosed          bool       `json:"is_closed"`
		}
		err := json.NewDecoder(res.Body).Decode(&sellResp)
		require.NoError(t, err)

		assert.Equal(t, 0.00, float64(sellResp.RemainingQuantity))
		assert.True(t, sellResp.IsClosed)

		// Verifica que ALUP11 não aparece mais na lista ativa
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/investments?telegram_id=%d", telegramID), nil)
		reqGet.Header.Set("X-Internal-Secret", app.InternalKey)
		resGet := app.ExecuteRequest(reqGet)
		require.Equal(t, http.StatusOK, resGet.Code)

		var listResp struct {
			Investments []struct {
				Ticker string `json:"ticker"`
			} `json:"investments"`
		}
		err = json.NewDecoder(resGet.Body).Decode(&listResp)
		require.NoError(t, err)

		assert.Len(t, listResp.Investments, 2)
		for _, inv := range listResp.Investments {
			assert.NotEqual(t, "ALUP11", inv.Ticker)
		}
	})

	// Cenário 6: Recompra Inicia Novo Ciclo Limpo (D-11)
	t.Run("Recompra Inicia Novo Ciclo Limpo", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "ALUP11",
			"quantity":    10,
			"price":       60.00,
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/investments", bytes.NewReader(body))
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		require.Equal(t, http.StatusOK, res.Code)

		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/investments?telegram_id=%d", telegramID), nil)
		reqGet.Header.Set("X-Internal-Secret", app.InternalKey)
		resGet := app.ExecuteRequest(reqGet)
		require.Equal(t, http.StatusOK, resGet.Code)

		var listResp struct {
			Investments []struct {
				Ticker       string     `json:"ticker"`
				Quantity     moneyFloat `json:"quantity"`
				AveragePrice moneyFloat `json:"average_price"`
				TotalCost    moneyFloat `json:"total_cost"`
			} `json:"investments"`
		}
		err := json.NewDecoder(resGet.Body).Decode(&listResp)
		require.NoError(t, err)

		var foundAlup bool
		for _, inv := range listResp.Investments {
			if inv.Ticker == "ALUP11" {
				foundAlup = true
				assert.Equal(t, 10.00, float64(inv.Quantity))
				assert.Equal(t, 60.00, float64(inv.AveragePrice)) // Novo ciclo limpo, PM estritamente 60.00
				assert.Equal(t, 600.00, float64(inv.TotalCost))
			}
		}
		assert.True(t, foundAlup)
	})

	// Cenário 7: Validações de Erro e Borda (T-03-02, T-03-03)
	t.Run("Validações de Erro e Borda", func(t *testing.T) {
		// Vender ticker inexistente
		bodyNotFound, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "VALE3",
			"quantity":    10,
		})
		req1 := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(bodyNotFound))
		req1.Header.Set("X-Internal-Secret", app.InternalKey)
		res1 := app.ExecuteRequest(req1)
		assert.Equal(t, http.StatusNotFound, res1.Code)
		assert.Contains(t, res1.Body.String(), "investimento não encontrado")

		// Vender quantidade maior que a custódia (PETR4 tem 50, tenta vender 100)
		bodyExcess, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "PETR4",
			"quantity":    100,
		})
		req2 := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(bodyExcess))
		req2.Header.Set("X-Internal-Secret", app.InternalKey)
		res2 := app.ExecuteRequest(req2)
		assert.Equal(t, http.StatusBadRequest, res2.Code)
		assert.Contains(t, res2.Body.String(), "quantidade insuficiente para venda")

		// Vender quantidade zero ou negativa
		bodyZero, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"ticker":      "PETR4",
			"quantity":    0,
		})
		req3 := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(bodyZero))
		req3.Header.Set("X-Internal-Secret", app.InternalKey)
		res3 := app.ExecuteRequest(req3)
		assert.Equal(t, http.StatusBadRequest, res3.Code)

		// Usuário inexistente
		bodyUserNotFound, _ := json.Marshal(map[string]any{
			"telegram_id": 99999999,
			"ticker":      "PETR4",
			"quantity":    10,
		})
		req4 := httptest.NewRequest(http.MethodPost, "/internal/investments/sell", bytes.NewReader(bodyUserNotFound))
		req4.Header.Set("X-Internal-Secret", app.InternalKey)
		res4 := app.ExecuteRequest(req4)
		assert.Equal(t, http.StatusNotFound, res4.Code)
		assert.Contains(t, res4.Body.String(), "Usuário não encontrado")
	})
}
