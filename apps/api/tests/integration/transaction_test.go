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

func TestTransactionAndBundlesFlow(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	user, token := app.CreateTestUser(t, 33445566, "Usuario Transacao")
	authHeader := "Bearer " + token
	catID := app.FindFirstCategory(t, user.ID)
	accountID := app.FindFirstBankAccount(t, user.ID)

	// 1. Criação de transação avulsa
	createBody, _ := json.Marshal(map[string]any{
		"description":     "Mercado Semanal",
		"amount":          250.75,
		"type":            "FLEXIBLE_EXPENSE",
		"due_date":        "2026-09-15",
		"category_id":     catID,
		"bank_account_id": accountID,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(createBody))
	createReq.Header.Set("Authorization", authHeader)
	createRes := app.ExecuteRequest(createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	var tx struct {
		ID          uuid.UUID `json:"id"`
		Description string    `json:"description"`
		Amount      float64   `json:"amount"`
		Type        string    `json:"type"`
		Status      string    `json:"status"`
		DueDate     string    `json:"due_date"`
	}
	err := json.NewDecoder(createRes.Body).Decode(&tx)
	require.NoError(t, err)
	assert.Equal(t, "Mercado Semanal", tx.Description)
	assert.Equal(t, "PROJECTED", tx.Status)
	assert.Equal(t, "2026-09-15", tx.DueDate)

	// 2. Atualiza a transação
	updateBody, _ := json.Marshal(map[string]any{
		"description":     "Mercado Semanal Atualizado",
		"amount":          280.00,
		"type":            "FLEXIBLE_EXPENSE",
		"due_date":        "2026-09-16",
		"category_id":     catID,
		"bank_account_id": accountID,
	})
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transactions/%s", tx.ID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Authorization", authHeader)
	updateRes := app.ExecuteRequest(updateReq)
	require.Equal(t, http.StatusOK, updateRes.Code)

	// 3. Criação de parcelamento (Bundle em 3x)
	bundleBody, _ := json.Marshal(map[string]any{
		"description":        "Notebook",
		"total_amount":       3000.00,
		"total_installments": 3,
		"first_due_date":     "2026-09-20",
		"category_id":        catID,
		"bank_account_id":    accountID,
	})
	bundleReq := httptest.NewRequest(http.MethodPost, "/transactions/bundles", bytes.NewReader(bundleBody))
	bundleReq.Header.Set("Authorization", authHeader)
	bundleRes := app.ExecuteRequest(bundleReq)
	require.Equal(t, http.StatusCreated, bundleRes.Code)

	var bundleResp struct {
		ID                uuid.UUID `json:"id"`
		Description       string    `json:"description"`
		TotalAmount       float64   `json:"total_amount"`
		TotalInstallments int       `json:"total_installments"`
		FirstDueDate      string    `json:"first_due_date"`
	}
	err = json.NewDecoder(bundleRes.Body).Decode(&bundleResp)
	require.NoError(t, err)
	assert.Equal(t, "Notebook", bundleResp.Description)
	assert.Equal(t, 3, bundleResp.TotalInstallments)

	// 4. Lista transações do usuário (deve conter a avulsa + 3 parcelas do bundle = 4)
	listReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRes := app.ExecuteRequest(listReq)
	require.Equal(t, http.StatusOK, listRes.Code)

	var list []struct {
		ID                uuid.UUID  `json:"id"`
		Description       string     `json:"description"`
		BundleID          *uuid.UUID `json:"bundle_id"`
		InstallmentNumber *int       `json:"installment_number"`
		Status            string     `json:"status"`
	}
	err = json.NewDecoder(listRes.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 4)

	// Identifica uma parcela do bundle
	var installmentID uuid.UUID
	for _, item := range list {
		if item.BundleID != nil {
			installmentID = item.ID
			assert.Equal(t, "COMMITTED", item.Status)
			break
		}
	}
	require.NotEqual(t, uuid.Nil, installmentID)

	// 5. Tenta alterar individualmente a parcela do bundle -> Deve ser rejeitada (400 Bad Request)
	instUpdateBody, _ := json.Marshal(map[string]any{
		"description": "Parcela Alterada",
		"amount":      999.00,
		"type":        "INSTALLMENT_EXPENSE",
		"due_date":    "2026-09-20",
		"category_id": catID,
	})
	instUpdateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/transactions/%s", installmentID), bytes.NewReader(instUpdateBody))
	instUpdateReq.Header.Set("Authorization", authHeader)
	instUpdateRes := app.ExecuteRequest(instUpdateReq)
	assert.Equal(t, http.StatusBadRequest, instUpdateRes.Code)

	var errResp struct {
		Errors []string `json:"errors"`
	}
	err = json.NewDecoder(instUpdateRes.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Errors[0], "parcelas de um parcelamento não podem ser alteradas individualmente")
}

func TestDailyCheckInAndSimulation(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	user, token := app.CreateTestUser(t, 44556677, "Usuario Checkin")
	authHeader := "Bearer " + token
	catID := app.FindFirstCategory(t, user.ID)

	// 1. Cria uma transação pendente (PROJECTED) para ser confirmada no check-in
	pendingBody, _ := json.Marshal(map[string]any{
		"description": "Internet",
		"amount":      120.00,
		"type":        "FIXED_EXPENSE",
		"due_date":    "2026-09-13",
		"category_id": catID,
	})
	pendingReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(pendingBody))
	pendingReq.Header.Set("Authorization", authHeader)
	pendingRes := app.ExecuteRequest(pendingReq)
	require.Equal(t, http.StatusCreated, pendingRes.Code)

	var pendingTx struct {
		ID uuid.UUID `json:"id"`
	}
	_ = json.NewDecoder(pendingRes.Body).Decode(&pendingTx)

	// 2. Executa o Check-in Diário
	checkInBody, _ := json.Marshal(map[string]any{
		"date":                "2026-09-13",
		"liquid_balance":      1500.00,
		"target_savings":      200.00,
		"flexible_budget_cap": 900.00,
		"untracked_expenses": []map[string]any{
			{"description": "Almoço", "amount": 35.50, "category_id": catID},
			{"description": "Café", "amount": 10.00, "category_id": catID},
		},
		"confirmed_pending_transaction_ids": []uuid.UUID{pendingTx.ID},
	})
	checkInReq := httptest.NewRequest(http.MethodPost, "/transactions/check-in", bytes.NewReader(checkInBody))
	checkInReq.Header.Set("Authorization", authHeader)
	checkInRes := app.ExecuteRequest(checkInReq)
	require.Equal(t, http.StatusOK, checkInRes.Code)

	var checkInResp struct {
		S2SCalculated        float64 `json:"s2s_calculated"`
		SpentToday           float64 `json:"spent_today"`
		DeltaFromSafeToSpend float64 `json:"delta_from_safe_to_spend"`
		HealthStatus         string  `json:"health_status"`
		NextDayS2S           float64 `json:"next_day_s2s"`
		DaysRemaining        int     `json:"days_remaining"`
	}
	err := json.NewDecoder(checkInRes.Body).Decode(&checkInResp)
	require.NoError(t, err)
	assert.Equal(t, 45.50, checkInResp.SpentToday)
	assert.Equal(t, 18, checkInResp.DaysRemaining)
	assert.True(t, checkInResp.S2SCalculated > 0)

	// 3. Executa segundo check-in no mesmo dia para verificar idempotência e upsert do snapshot
	checkInReq2 := httptest.NewRequest(http.MethodPost, "/transactions/check-in", bytes.NewReader(checkInBody))
	checkInReq2.Header.Set("Authorization", authHeader)
	checkInRes2 := app.ExecuteRequest(checkInReq2)
	require.Equal(t, http.StatusOK, checkInRes2.Code)

	// 4. Executa simulação What-If
	simBody, _ := json.Marshal(map[string]any{
		"date":                "2026-09-13",
		"liquid_balance":      1000.00,
		"target_savings":      100.00,
		"flexible_budget_cap": 900.00,
		"total_amount":        600.00,
		"installments":        6,
		"first_due_date":      "2026-09-20",
	})
	simReq := httptest.NewRequest(http.MethodPost, "/transactions/simulations", bytes.NewReader(simBody))
	simReq.Header.Set("Authorization", authHeader)
	simRes := app.ExecuteRequest(simReq)
	require.Equal(t, http.StatusOK, simRes.Code)

	var simResp struct {
		Cycles []struct {
			S2SToday     float64 `json:"s2s_today"`
			S2SReduction float64 `json:"s2s_reduction"`
			HealthStatus string  `json:"health_status"`
		} `json:"cycles"`
	}
	err = json.NewDecoder(simRes.Body).Decode(&simResp)
	require.NoError(t, err)
	assert.Len(t, simResp.Cycles, 6)
}
