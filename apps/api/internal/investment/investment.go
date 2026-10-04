package investment

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
)

var (
	ErrInvestmentNotFound   = errors.New("investimento não encontrado")
	ErrInvalidTicker        = errors.New("código do ativo inválido. Deve ter entre 1 e 12 caracteres alfanuméricos")
	ErrInvalidQuantity      = errors.New("quantidade deve ser maior que zero")
	ErrInvalidPrice         = errors.New("preço deve ser maior ou igual a zero")
	ErrInsufficientQuantity = errors.New("quantidade insuficiente para venda")
)

var tickerRegex = regexp.MustCompile(`^[A-Z0-9.\-_]{1,12}$`)

// Investment representa a posição consolidada de um ativo/ação do usuário.
type Investment struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id,omitempty"`
	Ticker       string          `json:"ticker"`
	Quantity     decimal.Decimal `json:"quantity"`
	AveragePrice money.Money     `json:"average_price"`
	TotalCost    money.Money     `json:"total_cost"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// NormalizeTicker valida e coloca em caixa alta o código do ativo (ex: "alup11 " -> "ALUP11").
func NormalizeTicker(ticker string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(ticker))
	if !tickerRegex.MatchString(normalized) {
		return "", ErrInvalidTicker
	}
	return normalized, nil
}

// CalculateNewAveragePrice calcula o Preço Médio Ponderado e a nova quantidade total após a compra de um novo lote.
// Fórmula: Novo PM = ((Qtd Atual * PM Atual) + (Qtd Nova * Preço Novo)) / (Qtd Total)
// Utiliza arredondamento Banker's Rounding (HalfEven) para 2 casas decimais no preço e 4 casas na quantidade.
func CalculateNewAveragePrice(
	currentQty decimal.Decimal,
	currentPM money.Money,
	newQty decimal.Decimal,
	newPrice money.Money,
) (decimal.Decimal, money.Money, error) {
	if !newQty.IsPositive() {
		return decimal.Zero, money.Zero(), ErrInvalidQuantity
	}
	if newPrice.IsNegative() {
		return decimal.Zero, money.Zero(), ErrInvalidPrice
	}

	if currentQty.IsZero() || currentQty.IsNegative() {
		return newQty.RoundBank(4), newPrice, nil
	}

	currentCost := currentQty.Mul(currentPM.Decimal())
	newCost := newQty.Mul(newPrice.Decimal())
	totalQty := currentQty.Add(newQty)
	totalCost := currentCost.Add(newCost)

	newPMDec := totalCost.Div(totalQty).RoundBank(2)
	newPM := money.New(newPMDec)

	return totalQty.RoundBank(4), newPM, nil
}

// CalculateSale calcula a quantidade remanescente após uma venda, mantendo o Preço Médio inalterado.
func CalculateSale(currentQty decimal.Decimal, sellQty decimal.Decimal) (remainingQty decimal.Decimal, isClosed bool, err error) {
	if !sellQty.IsPositive() {
		return decimal.Zero, false, ErrInvalidQuantity
	}
	if sellQty.GreaterThan(currentQty) {
		return decimal.Zero, false, ErrInsufficientQuantity
	}

	remaining := currentQty.Sub(sellQty).RoundBank(4)
	if remaining.IsZero() {
		return decimal.Zero, true, nil
	}

	return remaining, false, nil
}

// CalculateTotalCost calcula a posição financeira total (Qtd * PM) arredondada para Money.
func CalculateTotalCost(qty decimal.Decimal, avgPrice money.Money) money.Money {
	return money.New(qty.Mul(avgPrice.Decimal()).RoundBank(2))
}
