package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

const scale = 2

var (
	zeroDecimal = decimal.NewFromInt(0)
	hundred     = decimal.NewFromInt(100)
)

// Money encapsula quantias monetárias com precisão decimal exata e escala 2 (HalfEven).
type Money struct {
	val decimal.Decimal
}

// New cria um Money arredondado para escala 2 via HalfEven (Banker's rounding).
func New(d decimal.Decimal) Money {
	return Money{val: d.RoundBank(scale)}
}

// NewFromFloat cria um Money a partir de float64.
func NewFromFloat(f float64) Money {
	return New(decimal.NewFromFloat(f))
}

// NewFromString cria um Money a partir de string decimal (ex: "100.50").
func NewFromString(s string) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Money{}, err
	}
	return New(d), nil
}

// Zero retorna 0.00.
func Zero() Money {
	return Money{val: zeroDecimal}
}

// FromCents converte centavos inteiros (ex: 1050 cents = R$ 10.50).
func FromCents(cents int64) Money {
	return Money{val: decimal.New(cents, -scale)}
}

// ToCents retorna o valor total em centavos inteiros (ex: 10.50 -> 1050).
func (m Money) ToCents() int64 {
	return m.val.Shift(scale).IntPart()
}

// Decimal retorna o decimal.Decimal subjacente.
func (m Money) Decimal() decimal.Decimal {
	return m.val
}

// String formata com 2 casas decimais.
func (m Money) String() string {
	return m.val.StringFixed(scale)
}

// Add soma dois valores monetários.
func (m Money) Add(other Money) Money {
	return New(m.val.Add(other.val))
}

// Subtract subtrai outro valor.
func (m Money) Subtract(other Money) Money {
	return New(m.val.Sub(other.val))
}

// Multiply multiplica por um fator inteiro.
func (m Money) Multiply(factor int) Money {
	return New(m.val.Mul(decimal.NewFromInt(int64(factor))))
}

// IsNegative verifica se o valor é menor que zero.
func (m Money) IsNegative() bool {
	return m.val.IsNegative()
}

// IsPositive verifica se o valor é estritamente maior que zero.
func (m Money) IsPositive() bool {
	return m.val.IsPositive()
}

func (m Money) IsZero() bool {
	return m.val.IsZero()
}

// IsGreaterThan verifica se m > other.
func (m Money) IsGreaterThan(other Money) bool {
	return m.val.GreaterThan(other.val)
}

// IsLessThan verifica se m < other.
func (m Money) IsLessThan(other Money) bool {
	return m.val.LessThan(other.val)
}

// Compare compara m e other: -1 se m < other, 0 se m == other, 1 se m > other.
func (m Money) Compare(other Money) int {
	return m.val.Cmp(other.val)
}

// Min retorna o menor valor.
func Min(a, b Money) Money {
	if a.val.LessThanOrEqual(b.val) {
		return a
	}
	return b
}

// Max retorna o maior valor.
func Max(a, b Money) Money {
	if a.val.GreaterThanOrEqual(b.val) {
		return a
	}
	return b
}

// Sum soma uma lista de valores.
func Sum(values []Money) Money {
	total := Zero()
	for _, v := range values {
		total = total.Add(v)
	}
	return total
}

// Allocate divide o valor monetário em N parcelas usando RoundBank(2).
// A primeira parcela absorve o resto da divisão, garantindo soma exata: sum(parcelas) == total.
func (m Money) Allocate(parts int) ([]Money, error) {
	if parts <= 0 {
		return nil, errors.New("número de partes deve ser positivo")
	}

	partsDec := decimal.NewFromInt(int64(parts))
	partDec := m.val.Div(partsDec).RoundBank(scale)
	part := New(partDec)

	// remainder = total - (part * (parts - 1))
	otherPartsTotal := part.val.Mul(decimal.NewFromInt(int64(parts - 1)))
	remainder := New(m.val.Sub(otherPartsTotal))

	allocations := make([]Money, parts)
	allocations[0] = remainder
	for i := 1; i < parts; i++ {
		allocations[i] = part
	}
	return allocations, nil
}

// RatioOver calcula a razão m / base com 6 casas decimais e RoundBank.
func (m Money) RatioOver(base Money) decimal.Decimal {
	if base.IsZero() {
		return zeroDecimal
	}
	return m.val.Div(base.val).RoundBank(6)
}

// PercentOver calcula a porcentagem (m / base) * 100 com escala 2 e RoundBank.
func (m Money) PercentOver(base Money) decimal.Decimal {
	if base.IsZero() {
		return zeroDecimal
	}
	return m.RatioOver(base).Mul(hundred).RoundBank(scale)
}

// MarshalJSON serializa como número decimal JSON.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalJSON desserializa a partir de número ou string JSON.
func (m *Money) UnmarshalJSON(data []byte) error {
	var d decimal.Decimal
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	*m = New(d)
	return nil
}

// Value implementa driver.Valuer para gravação no PostgreSQL NUMERIC(19, 2).
func (m Money) Value() (driver.Value, error) {
	return m.String(), nil
}

// Scan implementa sql.Scanner para leitura do PostgreSQL.
func (m *Money) Scan(value any) error {
	if value == nil {
		*m = Zero()
		return nil
	}
	switch v := value.(type) {
	case float64:
		*m = NewFromFloat(v)
		return nil
	case string:
		d, err := decimal.NewFromString(v)
		if err != nil {
			return err
		}
		*m = New(d)
		return nil
	case []byte:
		d, err := decimal.NewFromString(string(v))
		if err != nil {
			return err
		}
		*m = New(d)
		return nil
	default:
		return fmt.Errorf("tipo não suportado para Money: %T", value)
	}
}
