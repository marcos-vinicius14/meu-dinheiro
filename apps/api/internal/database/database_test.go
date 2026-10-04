package database_test

import (
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL normal sem alteracao",
			input:    "postgres://postgres:minhasenha@postgres:5432/meu_dinheiro?sslmode=disable",
			expected: "postgres://postgres:minhasenha@postgres:5432/meu_dinheiro?sslmode=disable",
		},
		{
			name:     "URL com espacos nas pontas",
			input:    "   postgres://postgres:minhasenha@postgres:5432/meu_dinheiro?sslmode=disable \n",
			expected: "postgres://postgres:minhasenha@postgres:5432/meu_dinheiro?sslmode=disable",
		},
		{
			name:     "Senha com espacos no meio",
			input:    "postgres://postgres:minha senha forte@postgres:5432/meu_dinheiro?sslmode=disable",
			expected: "postgres://postgres:minha%20senha%20forte@postgres:5432/meu_dinheiro?sslmode=disable",
		},
		{
			name:     "Senha com espaco no final",
			input:    "postgres://postgres:minhasenha @postgres:5432/meu_dinheiro?sslmode=disable",
			expected: "postgres://postgres:minhasenha%20@postgres:5432/meu_dinheiro?sslmode=disable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := database.SanitizeDatabaseURL(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMaskDatabaseURL(t *testing.T) {
	urlWithSpaces := "postgres://postgres:minha senha forte@postgres:5432/meu_dinheiro?sslmode=disable"
	masked := database.MaskDatabaseURL(urlWithSpaces)
	assert.Equal(t, "postgres://postgres:%2A%2A%2A%2A@postgres:5432/meu_dinheiro?sslmode=disable", masked)
}
