package investment_test

import (
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/investment"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(s string) decimal.Decimal {
	val, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return val
}

func m(s string) money.Money {
	val, err := money.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return val
}

func TestNormalizeTicker(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		shouldError bool
	}{
		{"alup11", "ALUP11", false},
		{" BBAS3 ", "BBAS3", false},
		{"ivvb11", "IVVB11", false},
		{"btc", "BTC", false},
		{"", "", true},
		{"INVALID TICKER VERY LONG", "", true},
		{"PETR4@", "", true},
	}

	for _, tt := range tests {
		res, err := investment.NormalizeTicker(tt.input)
		if tt.shouldError {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.expected, res)
		}
	}
}

func TestCalculateNewAveragePriceInitialLot(t *testing.T) {
	qty, pm, err := investment.CalculateNewAveragePrice(d("0"), money.Zero(), d("10"), m("42.23"))
	require.NoError(t, err)
	assert.Equal(t, "10", qty.String())
	assert.Equal(t, "42.23", pm.String())
}

func TestCalculateNewAveragePriceIncrementalPurchase(t *testing.T) {
	// Lote 1: 10 ações a R$ 40,00 (Total R$ 400,00)
	// Lote 2: 10 ações a R$ 50,00 (Total R$ 500,00)
	// Total: 20 ações por R$ 900,00 -> Novo PM: R$ 45,00
	qty, pm, err := investment.CalculateNewAveragePrice(d("10"), m("40.00"), d("10"), m("50.00"))
	require.NoError(t, err)
	assert.Equal(t, "20", qty.String())
	assert.Equal(t, "45.00", pm.String())
}

func TestCalculateNewAveragePriceWeightedRounding(t *testing.T) {
	// Lote 1: 10 ações a R$ 42,23 (Total R$ 422,30)
	// Lote 2: 5 ações a R$ 45,10 (Total R$ 225,50)
	// Total: 15 ações por R$ 647,80 -> PM: 647.80 / 15 = 43.18666... -> Banker's round: 43.19
	qty, pm, err := investment.CalculateNewAveragePrice(d("10"), m("42.23"), d("5"), m("45.10"))
	require.NoError(t, err)
	assert.Equal(t, "15", qty.String())
	assert.Equal(t, "43.19", pm.String())
}

func TestCalculateNewAveragePriceRejectsInvalidInput(t *testing.T) {
	// Quantidade zero ou negativa
	_, _, err := investment.CalculateNewAveragePrice(d("10"), m("40.00"), d("0"), m("50.00"))
	assert.ErrorIs(t, err, investment.ErrInvalidQuantity)

	_, _, err = investment.CalculateNewAveragePrice(d("10"), m("40.00"), d("-5"), m("50.00"))
	assert.ErrorIs(t, err, investment.ErrInvalidQuantity)

	// Preço negativo
	negPrice, _ := money.NewFromString("-10.00")
	_, _, err = investment.CalculateNewAveragePrice(d("10"), m("40.00"), d("5"), negPrice)
	assert.ErrorIs(t, err, investment.ErrInvalidPrice)
}

func TestCalculateSale(t *testing.T) {
	// Venda parcial: tem 10, vende 4 -> resta 6, isClosed = false
	remaining, closed, err := investment.CalculateSale(d("10"), d("4"))
	require.NoError(t, err)
	assert.Equal(t, "6", remaining.String())
	assert.False(t, closed)

	// Venda total: tem 10, vende 10 -> resta 0, isClosed = true
	remaining, closed, err = investment.CalculateSale(d("10"), d("10"))
	require.NoError(t, err)
	assert.Equal(t, "0", remaining.String())
	assert.True(t, closed)

	// Venda superior ao saldo -> erro
	_, _, err = investment.CalculateSale(d("10"), d("15"))
	assert.ErrorIs(t, err, investment.ErrInsufficientQuantity)

	// Venda zero ou negativa -> erro
	_, _, err = investment.CalculateSale(d("10"), d("0"))
	assert.ErrorIs(t, err, investment.ErrInvalidQuantity)
}
