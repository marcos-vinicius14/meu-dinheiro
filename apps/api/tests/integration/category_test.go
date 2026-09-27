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

func TestCategoryCRUDAndUniqueness(t *testing.T) {
	app := integration.SetupTestApp(t)
	defer app.CleanDatabase(t)

	user, token := app.CreateTestUser(t, 22334455, "Usuario Categoria")
	authHeader := "Bearer " + token

	// 1. Cria nova categoria
	icon := "book"
	createBody, _ := json.Marshal(map[string]any{
		"description": "Educação",
		"icon":        &icon,
		"is_flexible": false,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(createBody))
	createReq.Header.Set("Authorization", authHeader)
	createRes := app.ExecuteRequest(createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	var cat struct {
		ID          uuid.UUID `json:"id"`
		Description string    `json:"description"`
		Icon        *string   `json:"icon"`
		IsFlexible  bool      `json:"is_flexible"`
	}
	err := json.NewDecoder(createRes.Body).Decode(&cat)
	require.NoError(t, err)
	assert.Equal(t, "Educação", cat.Description)
	assert.Equal(t, "book", *cat.Icon)
	assert.False(t, cat.IsFlexible)

	// 2. Tenta criar categoria duplicada (case-insensitive: "educação") -> Deve ser rejeitada
	dupBody, _ := json.Marshal(map[string]any{
		"description": "educação",
	})
	dupReq := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(dupBody))
	dupReq.Header.Set("Authorization", authHeader)
	dupRes := app.ExecuteRequest(dupReq)
	assert.Equal(t, http.StatusBadRequest, dupRes.Code)

	var errResp struct {
		Errors []string `json:"errors"`
	}
	err = json.NewDecoder(dupRes.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Errors[0], "categoria já cadastrada")

	// 3. Atualiza a categoria
	newIcon := "graduation-cap"
	updateBody, _ := json.Marshal(map[string]any{
		"description": "Cursos & Livros",
		"icon":        &newIcon,
	})
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%s", cat.ID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Authorization", authHeader)
	updateRes := app.ExecuteRequest(updateReq)
	require.Equal(t, http.StatusOK, updateRes.Code)

	// 4. Cria transação vinculada e tenta deletar -> Deve retornar 409 Conflict
	txBody, _ := json.Marshal(map[string]any{
		"description": "Curso Go",
		"amount":      150.00,
		"type":        "FIXED_EXPENSE",
		"due_date":    "2026-09-25",
		"category_id": cat.ID,
	})
	txReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(txBody))
	txReq.Header.Set("Authorization", authHeader)
	txRes := app.ExecuteRequest(txReq)
	require.Equal(t, http.StatusCreated, txRes.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%s", cat.ID), nil)
	deleteReq.Header.Set("Authorization", authHeader)
	deleteRes := app.ExecuteRequest(deleteReq)
	assert.Equal(t, http.StatusConflict, deleteRes.Code)

	// 5. Cria outra categoria avulsa e deleta -> Sucesso 204
	avulsaBody, _ := json.Marshal(map[string]any{"description": "Temporária"})
	avulsaReq := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(avulsaBody))
	avulsaReq.Header.Set("Authorization", authHeader)
	avulsaRes := app.ExecuteRequest(avulsaReq)
	require.Equal(t, http.StatusCreated, avulsaRes.Code)

	var avulsaCat struct {
		ID uuid.UUID `json:"id"`
	}
	_ = json.NewDecoder(avulsaRes.Body).Decode(&avulsaCat)

	deleteAvulsaReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%s", avulsaCat.ID), nil)
	deleteAvulsaReq.Header.Set("Authorization", authHeader)
	deleteAvulsaRes := app.ExecuteRequest(deleteAvulsaReq)
	assert.Equal(t, http.StatusNoContent, deleteAvulsaRes.Code)
	_ = user
}
