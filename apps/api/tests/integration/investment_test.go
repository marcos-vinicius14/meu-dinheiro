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

func TestInvestmentCRUDAndWeightedAveragePrice(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	_, token := app.CreateTestUser(t, 99887766, "Investidor Teste")
	authHeader := "Bearer " + token

	// 1. Cadastra o primeiro lote de ALUP11 (10 ações a R$ 40,00)
	body1, _ := json.Marshal(map[string]any{
		"ticker":   "alup11",
		"quantity": 10,
		"price":    40.00,
	})
	req1 := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewReader(body1))
	req1.Header.Set("Authorization", authHeader)
	res1 := app.ExecuteRequest(req1)
	require.Equal(t, http.StatusCreated, res1.Code)

	var inv1 struct {
		ID           uuid.UUID `json:"id"`
		Ticker       string    `json:"ticker"`
		Quantity     string    `json:"quantity"`
		AveragePrice float64   `json:"average_price"`
		TotalCost    float64   `json:"total_cost"`
	}
	err := json.NewDecoder(res1.Body).Decode(&inv1)
	require.NoError(t, err)
	assert.Equal(t, "ALUP11", inv1.Ticker)
	assert.Equal(t, "10", inv1.Quantity)
	assert.Equal(t, 40.00, inv1.AveragePrice)
	assert.Equal(t, 400.00, inv1.TotalCost)

	// 2. Adiciona segundo lote de ALUP11 (10 ações a R$ 50,00) -> Novo PM deve ser R$ 45,00
	body2, _ := json.Marshal(map[string]any{
		"ticker":   "ALUP11",
		"quantity": 10,
		"price":    50.00,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewReader(body2))
	req2.Header.Set("Authorization", authHeader)
	res2 := app.ExecuteRequest(req2)
	require.Equal(t, http.StatusCreated, res2.Code)

	var inv2 struct {
		ID           uuid.UUID `json:"id"`
		Ticker       string    `json:"ticker"`
		Quantity     string    `json:"quantity"`
		AveragePrice float64   `json:"average_price"`
		TotalCost    float64   `json:"total_cost"`
	}
	err = json.NewDecoder(res2.Body).Decode(&inv2)
	require.NoError(t, err)
	assert.Equal(t, inv1.ID, inv2.ID) // mesmo ativo atualizado
	assert.Equal(t, "20", inv2.Quantity)
	assert.Equal(t, 45.00, inv2.AveragePrice)
	assert.Equal(t, 900.00, inv2.TotalCost)

	// 3. Cadastra segundo ativo: BBAS3 (50 ações a R$ 27,50) -> Total R$ 1.375,00
	body3, _ := json.Marshal(map[string]any{
		"ticker":   "BBAS3",
		"quantity": 50,
		"price":    27.50,
	})
	req3 := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewReader(body3))
	req3.Header.Set("Authorization", authHeader)
	res3 := app.ExecuteRequest(req3)
	require.Equal(t, http.StatusCreated, res3.Code)

	// 4. Lista os investimentos do usuário
	listReq := httptest.NewRequest(http.MethodGet, "/investments", nil)
	listReq.Header.Set("Authorization", authHeader)
	listRes := app.ExecuteRequest(listReq)
	require.Equal(t, http.StatusOK, listRes.Code)

	var portfolio struct {
		Investments []struct {
			ID     uuid.UUID `json:"id"`
			Ticker string    `json:"ticker"`
		} `json:"investments"`
		TotalInvested float64 `json:"total_invested"`
	}
	err = json.NewDecoder(listRes.Body).Decode(&portfolio)
	require.NoError(t, err)
	assert.Len(t, portfolio.Investments, 2)
	assert.Equal(t, 2275.00, portfolio.TotalInvested) // 900 + 1375

	// 5. Deleta o ativo ALUP11
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/investments/%s", inv1.ID), nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRes := app.ExecuteRequest(deleteReq)
	assert.Equal(t, http.StatusNoContent, deleteRes.Code)

	// 6. Confirma que agora só resta 1 ativo
	listRes2 := app.ExecuteRequest(listReq)
	var portfolio2 struct {
		Investments   []any   `json:"investments"`
		TotalInvested float64 `json:"total_invested"`
	}
	_ = json.NewDecoder(listRes2.Body).Decode(&portfolio2)
	assert.Len(t, portfolio2.Investments, 1)
	assert.Equal(t, 1375.00, portfolio2.TotalInvested)
}
