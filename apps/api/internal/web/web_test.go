package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckGetAndHead(t *testing.T) {
	r := web.NewRouter()
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, http.StatusOK, map[string]string{"status": "UP"})
	}
	r.Get("/health", healthHandler)
	r.Head("/health", healthHandler)

	// Test GET
	reqGet := httptest.NewRequest(http.MethodGet, "/health", nil)
	rrGet := httptest.NewRecorder()
	r.ServeHTTP(rrGet, reqGet)
	assert.Equal(t, http.StatusOK, rrGet.Code)
	assert.Contains(t, rrGet.Body.String(), `"status":"UP"`)

	// Test HEAD
	reqHead := httptest.NewRequest(http.MethodHead, "/health", nil)
	rrHead := httptest.NewRecorder()
	r.ServeHTTP(rrHead, reqHead)
	assert.Equal(t, http.StatusOK, rrHead.Code)
}
