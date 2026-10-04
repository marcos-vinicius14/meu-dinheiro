package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalBotAPI_Predictive(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	telegramID := int64(77889900)

	// 1. Cria usuário e setup financeiro via Onboarding
	onboardingBody, _ := json.Marshal(map[string]any{
		"telegram_id":     telegramID,
		"first_name":      "Pred User",
		"username":        "pred_user",
		"initial_balance": 5000.00,
		"cycle_start_day": 5,
		"fixed_expenses": []map[string]any{
			{"description": "Aluguel", "amount": 1500.00, "category_name": "Moradia"},
			{"description": "Supermercado", "amount": 800.00, "category_name": "Alimentação"},
		},
		"emergency_fund_months": 6,
		"target_savings":        500.00,
		"flexible_budget_cap":   3000.00,
		"investments": []map[string]any{
			{"ticker": "ITSA4", "quantity": 100, "average_price": 10.50},
		},
	})

	onbReq := httptest.NewRequest(http.MethodPost, "/internal/users/onboarding", bytes.NewReader(onboardingBody))
	onbReq.Header.Set("X-Internal-Secret", app.InternalKey)
	onbRes := app.ExecuteRequest(onbReq)
	require.Equal(t, http.StatusOK, onbRes.Code)

	// 2. Segurança: chamadas sem X-Internal-Secret devem retornar 403 Forbidden
	t.Run("Segurança: acesso restrito a chave interna", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/transactions/simulations", bytes.NewReader([]byte("{}")))
		res := app.ExecuteRequest(req)
		assert.Equal(t, http.StatusForbidden, res.Code)

		req2 := httptest.NewRequest(http.MethodPost, "/internal/transactions/checkin", bytes.NewReader([]byte("{}")))
		res2 := app.ExecuteRequest(req2)
		assert.Equal(t, http.StatusForbidden, res2.Code)

		req3 := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/balance", bytes.NewReader([]byte("{}")))
		res3 := app.ExecuteRequest(req3)
		assert.Equal(t, http.StatusForbidden, res3.Code)
	})

	// 3. Simulação What-If de Compra Parcelada (POST /internal/transactions/simulations)
	t.Run("Simulação What-If de Compra Parcelada e À Vista", func(t *testing.T) {
		// 3a. Compra Parcelada em 12x
		simBody, _ := json.Marshal(map[string]any{
			"telegram_id":  telegramID,
			"amount":       2400.00,
			"installments": 12,
		})

		simReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/simulations", bytes.NewReader(simBody))
		simReq.Header.Set("X-Internal-Secret", app.InternalKey)
		simRes := app.ExecuteRequest(simReq)
		require.Equal(t, http.StatusOK, simRes.Code)

		var simResp struct {
			TotalAmount                  float64 `json:"total_amount"`
			Installments                 int     `json:"installments"`
			InstallmentAmount            float64 `json:"installment_amount"`
			CurrentCycleS2SBefore        float64 `json:"current_cycle_s2s_before"`
			CurrentCycleS2SAfter         float64 `json:"current_cycle_s2s_after"`
			CurrentCycleReduction        float64 `json:"current_cycle_reduction"`
			CurrentCycleReductionPercent string  `json:"current_cycle_reduction_percent"`
			CriticalCycleNumber          int     `json:"critical_cycle_number"`
			CriticalCycleS2S             float64 `json:"critical_cycle_s2s"`
			CriticalCycleHealthStatus    string  `json:"critical_cycle_health_status"`
			DeficitRiskAlert             bool    `json:"deficit_risk_alert"`
			RecommendationMessage        string  `json:"recommendation_message"`
			Cycles                       []any   `json:"cycles"`
		}

		err := json.NewDecoder(simRes.Body).Decode(&simResp)
		require.NoError(t, err)
		assert.Equal(t, 2400.00, simResp.TotalAmount)
		assert.Equal(t, 12, simResp.Installments)
		assert.Equal(t, 200.00, simResp.InstallmentAmount)
		assert.True(t, simResp.CurrentCycleS2SBefore > 0)
		assert.True(t, simResp.CurrentCycleS2SAfter < simResp.CurrentCycleS2SBefore)
		assert.True(t, simResp.CriticalCycleNumber >= 1)
		assert.NotEmpty(t, simResp.CriticalCycleHealthStatus)
		assert.NotEmpty(t, simResp.RecommendationMessage)
		assert.Len(t, simResp.Cycles, 12)

		// 3b. Compra À Vista (installments default = 1)
		simCashBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"amount":      350.00,
		})
		simCashReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/simulations", bytes.NewReader(simCashBody))
		simCashReq.Header.Set("X-Internal-Secret", app.InternalKey)
		simCashRes := app.ExecuteRequest(simCashReq)
		require.Equal(t, http.StatusOK, simCashRes.Code)

		var simCashResp struct {
			TotalAmount       float64 `json:"total_amount"`
			Installments      int     `json:"installments"`
			InstallmentAmount float64 `json:"installment_amount"`
		}
		err = json.NewDecoder(simCashRes.Body).Decode(&simCashResp)
		require.NoError(t, err)
		assert.Equal(t, 350.00, simCashResp.TotalAmount)
		assert.Equal(t, 1, simCashResp.Installments)
		assert.Equal(t, 350.00, simCashResp.InstallmentAmount)

		// 3c. Validação: amount <= 0 deve retornar 400 Bad Request
		invalidSimBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"amount":      -10.00,
		})
		invSimReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/simulations", bytes.NewReader(invalidSimBody))
		invSimReq.Header.Set("X-Internal-Secret", app.InternalKey)
		invSimRes := app.ExecuteRequest(invSimReq)
		assert.Equal(t, http.StatusBadRequest, invSimRes.Code)
	})

	// 4. Check-in Diário Idempotente (POST /internal/transactions/checkin)
	t.Run("Check-in Diário Guiado com Idempotência (UPSERT)", func(t *testing.T) {
		todayStr := time.Now().UTC().Format("2006-01-02")

		// 4a. Primeiro check-in do dia com 1 despesa não registrada (Lanche R$ 35,00)
		checkinBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"date":        todayStr,
			"untracked_expenses": []map[string]any{
				{
					"description":   "Lanche da Tarde",
					"amount":        35.00,
					"category_name": "Alimentação",
				},
			},
		})

		chkReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/checkin", bytes.NewReader(checkinBody))
		chkReq.Header.Set("X-Internal-Secret", app.InternalKey)
		chkRes := app.ExecuteRequest(chkReq)
		require.Equal(t, http.StatusOK, chkRes.Code)

		var chkResp struct {
			S2SCalculated moneyFloat `json:"s2s_calculated"`
			SpentToday    moneyFloat `json:"spent_today"`
			DailyQuota    moneyFloat `json:"daily_quota"`
			DeltaSavings  moneyFloat `json:"delta_savings"`
			HealthStatus  string     `json:"health_status"`
			NextDayS2S    moneyFloat `json:"next_day_s2s"`
			DaysRemaining int        `json:"days_remaining"`
			Message       string     `json:"message"`
		}

		err := json.NewDecoder(chkRes.Body).Decode(&chkResp)
		require.NoError(t, err)
		assert.Equal(t, 35.00, float64(chkResp.SpentToday))
		assert.True(t, float64(chkResp.S2SCalculated) > 0)
		assert.NotEmpty(t, chkResp.HealthStatus)
		assert.NotEmpty(t, chkResp.Message)

		// Verifica no banco de dados real (Postgres) se o snapshot foi persistido em tb_check_in_snapshots
		var countSnapshots int
		normToday := dateinterval.NormalizeDate(time.Now().UTC())
		err = app.Pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM tb_check_in_snapshots WHERE check_in_date = $1", normToday,
		).Scan(&countSnapshots)
		require.NoError(t, err)
		assert.Equal(t, 1, countSnapshots)

		// 4b. Idempotência (UPSERT): Segundo check-in na mesma data com novo gasto adicional (Café R$ 15,00)
		checkinBody2, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"date":        todayStr,
			"untracked_expenses": []map[string]any{
				{
					"description":   "Café Noturno",
					"amount":        15.00,
					"category_name": "Alimentação",
				},
			},
		})

		chkReq2 := httptest.NewRequest(http.MethodPost, "/internal/transactions/checkin", bytes.NewReader(checkinBody2))
		chkReq2.Header.Set("X-Internal-Secret", app.InternalKey)
		chkRes2 := app.ExecuteRequest(chkReq2)
		require.Equal(t, http.StatusOK, chkRes2.Code)

		var chkResp2 struct {
			SpentToday float64 `json:"spent_today"`
		}
		err = json.NewDecoder(chkRes2.Body).Decode(&chkResp2)
		require.NoError(t, err)
		// O total gasto hoje agora inclui 35.00 + 15.00 = 50.00
		assert.Equal(t, 50.00, chkResp2.SpentToday)

		// O banco deve continuar com exatamente 1 registro para essa data (sem duplicatas)
		err = app.Pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM tb_check_in_snapshots WHERE check_in_date = $1", normToday,
		).Scan(&countSnapshots)
		require.NoError(t, err)
		assert.Equal(t, 1, countSnapshots)
	})

	// 5. Ajuste de Saldo Bancário em Conta Corrente (POST /internal/bank-accounts/balance)
	t.Run("Ajuste de Saldo de Conta Bancária", func(t *testing.T) {
		adjBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"new_balance": 4800.00,
		})

		adjReq := httptest.NewRequest(http.MethodPost, "/internal/bank-accounts/balance", bytes.NewReader(adjBody))
		adjReq.Header.Set("X-Internal-Secret", app.InternalKey)
		adjRes := app.ExecuteRequest(adjReq)
		require.Equal(t, http.StatusOK, adjRes.Code)

		var adjResp struct {
			AccountName        string  `json:"account_name"`
			PreviousBalance    float64 `json:"previous_balance"`
			NewBalance         float64 `json:"new_balance"`
			TotalLiquidBalance float64 `json:"total_liquid_balance"`
			Message            string  `json:"message"`
		}

		err := json.NewDecoder(adjRes.Body).Decode(&adjResp)
		require.NoError(t, err)
		assert.Equal(t, 4800.00, adjResp.NewBalance)
		assert.Equal(t, 4750.00, adjResp.TotalLiquidBalance)
		assert.NotEmpty(t, adjResp.Message)

		// Consulta o contexto via GET /internal/users/context-by-telegram e valida sincronia
		ctxReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/internal/users/context-by-telegram?telegram_id=%d", telegramID), nil)
		ctxReq.Header.Set("X-Internal-Secret", app.InternalKey)
		ctxRes := app.ExecuteRequest(ctxReq)
		require.Equal(t, http.StatusOK, ctxRes.Code)

		var ctxResp struct {
			TotalLiquidBalance float64 `json:"total_liquid_balance"`
		}
		err = json.NewDecoder(ctxRes.Body).Decode(&ctxResp)
		require.NoError(t, err)
		assert.Equal(t, 4750.00, ctxResp.TotalLiquidBalance)
	})

	// 6. Registro de Entrada de Dinheiro (POST /internal/transactions/income)
	t.Run("Registro de Receita com Recalibração de S2S e Saldo Líquido", func(t *testing.T) {
		incBody, _ := json.Marshal(map[string]any{
			"telegram_id": telegramID,
			"amount":      2000.00,
			"description": "Salário Mensal",
		})

		incReq := httptest.NewRequest(http.MethodPost, "/internal/transactions/income", bytes.NewReader(incBody))
		incReq.Header.Set("X-Internal-Secret", app.InternalKey)
		incRes := app.ExecuteRequest(incReq)
		require.Equal(t, http.StatusCreated, incRes.Code)

		var incResp struct {
			Transaction struct {
				ID          string     `json:"id"`
				Description string     `json:"description"`
				Amount      moneyFloat `json:"amount"`
				Type        string     `json:"type"`
				Status      string     `json:"status"`
			} `json:"transaction"`
			CategoryName       string     `json:"category_name"`
			PreviousS2S        moneyFloat `json:"previous_s2s"`
			NewS2S             moneyFloat `json:"new_s2s"`
			HealthStatus       string     `json:"health_status"`
			DaysRemaining      int        `json:"days_remaining"`
			TotalLiquidBalance moneyFloat `json:"total_liquid_balance"`
		}

		err := json.NewDecoder(incRes.Body).Decode(&incResp)
		require.NoError(t, err)
		assert.Equal(t, "Salário Mensal", incResp.Transaction.Description)
		assert.Equal(t, 2000.00, float64(incResp.Transaction.Amount))
		assert.Equal(t, "INCOME", incResp.Transaction.Type)
		assert.Equal(t, "CONFIRMED", incResp.Transaction.Status)
		assert.Equal(t, 6750.00, float64(incResp.TotalLiquidBalance))
		assert.True(t, incResp.NewS2S >= incResp.PreviousS2S)
		assert.NotEmpty(t, incResp.HealthStatus)
	})

	// 7. Usuário Inexistente deve retornar 404 Not Found
	t.Run("Usuário não encontrado retorna 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"telegram_id": int64(9999999999),
			"amount":      100.00,
		})
		req := httptest.NewRequest(http.MethodPost, "/internal/transactions/simulations", bytes.NewReader(body))
		req.Header.Set("X-Internal-Secret", app.InternalKey)
		res := app.ExecuteRequest(req)
		assert.Equal(t, http.StatusNotFound, res.Code)
	})
}

// moneyFloat auxilia na desserialização de valores que podem vir como float ou money.Money
type moneyFloat float64

func (m *moneyFloat) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err == nil {
		*m = moneyFloat(f)
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		var val float64
		_, _ = fmt.Sscanf(str, "%f", &val)
		*m = moneyFloat(val)
		return nil
	}
	return nil
}
