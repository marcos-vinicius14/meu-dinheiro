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

func TestBankAccountCRUDAndLimits(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	// Cria usuário de teste (já provisiona 1 conta bancária padrão "Conta Corrente")
	user, token := app.CreateTestUser(t, 12345678, "Usuario Teste")

	authHeader := "Bearer " + token

	// 1. Cria a 2ª conta bancária
	body2, _ := json.Marshal(map[string]any{
		"name":            "Poupança",
		"type":            "SAVINGS",
		"initial_balance": 500.00,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/bank-accounts", bytes.NewReader(body2))
	req2.Header.Set("Authorization", authHeader)
	res2 := app.ExecuteRequest(req2)
	require.Equal(t, http.StatusCreated, res2.Code)

	var account2 struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Type           string    `json:"type"`
		InitialBalance float64   `json:"initial_balance"`
	}
	err := json.NewDecoder(res2.Body).Decode(&account2)
	require.NoError(t, err)
	assert.Equal(t, "Poupança", account2.Name)
	assert.Equal(t, "SAVINGS", account2.Type)

	// 2. Cria a 3ª conta bancária
	body3, _ := json.Marshal(map[string]any{
		"name":            "Investimentos",
		"type":            "INVESTMENT",
		"initial_balance": 1000.00,
	})
	req3 := httptest.NewRequest(http.MethodPost, "/bank-accounts", bytes.NewReader(body3))
	req3.Header.Set("Authorization", authHeader)
	res3 := app.ExecuteRequest(req3)
	require.Equal(t, http.StatusCreated, res3.Code)

	// 3. Tenta criar a 4ª conta bancária (deve ser rejeitada pelo limite de 3 contas)
	body4, _ := json.Marshal(map[string]any{
		"name": "Conta Extra",
		"type": "CHECKING",
	})
	req4 := httptest.NewRequest(http.MethodPost, "/bank-accounts", bytes.NewReader(body4))
	req4.Header.Set("Authorization", authHeader)
	res4 := app.ExecuteRequest(req4)
	assert.Equal(t, http.StatusBadRequest, res4.Code)

	var errResp struct {
		Errors []string `json:"errors"`
	}
	err = json.NewDecoder(res4.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Errors[0], "limite de 3 contas bancárias atingido")

	// 4. Lista as contas do usuário (deve ter 3)
	listReq := httptest.NewRequest(http.MethodGet, "/bank-accounts", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRes := app.ExecuteRequest(listReq)
	require.Equal(t, http.StatusOK, listRes.Code)

	var list []struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	err = json.NewDecoder(listRes.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// 5. Atualiza a conta 2
	updateBody, _ := json.Marshal(map[string]any{
		"name":            "Poupança Ouro",
		"initial_balance": 750.50,
	})
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/bank-accounts/%s", account2.ID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Authorization", authHeader)
	updateRes := app.ExecuteRequest(updateReq)
	require.Equal(t, http.StatusOK, updateRes.Code)

	// 6. Vincula uma transação à conta 2 e tenta deletar (deve dar 409 Conflict)
	catID := app.FindFirstCategory(t, user.ID)
	txBody, _ := json.Marshal(map[string]any{
		"description":     "Depósito",
		"amount":          100.00,
		"type":            "INCOME",
		"due_date":        "2026-09-20",
		"category_id":     catID,
		"bank_account_id": account2.ID,
	})
	txReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(txBody))
	txReq.Header.Set("Authorization", authHeader)
	txRes := app.ExecuteRequest(txReq)
	require.Equal(t, http.StatusCreated, txRes.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/bank-accounts/%s", account2.ID), nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRes := app.ExecuteRequest(deleteReq)
	assert.Equal(t, http.StatusConflict, deleteRes.Code)

	// 7. Deleta a conta 3 (sem transações vinculadas) -> Sucesso 204
	var account3 struct {
		ID uuid.UUID `json:"id"`
	}
	_ = json.NewDecoder(res3.Body).Decode(&account3)

	delete3Req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/bank-accounts/%s", account3.ID), nil)
	delete3Req.Header.Set("Authorization", authHeader)
	delete3Res := app.ExecuteRequest(delete3Req)
	assert.Equal(t, http.StatusNoContent, delete3Res.Code)
}
