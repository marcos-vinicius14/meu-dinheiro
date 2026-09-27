package money_test

import (
	"encoding/json"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustMoney(s string) money.Money {
	m, err := money.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return m
}

func TestAddSumsValues(t *testing.T) {
	result := mustMoney("10.50").Add(mustMoney("2.25"))
	assert.Equal(t, "12.75", result.String())
}

func TestSubtractReturnsDifference(t *testing.T) {
	result := mustMoney("10.00").Subtract(mustMoney("3.50"))
	assert.Equal(t, "6.50", result.String())
}

func TestMultiplyScalesValue(t *testing.T) {
	result := mustMoney("10.00").Multiply(3)
	assert.Equal(t, "30.00", result.String())
}

func TestMultiplyByZeroReturnsZero(t *testing.T) {
	result := mustMoney("10.00").Multiply(0)
	assert.True(t, result.IsZero())
	assert.Equal(t, "0.00", result.String())
}

func TestAllocateSplitsEvenlyWhenDivisible(t *testing.T) {
	parts, err := mustMoney("90.00").Allocate(3)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	assert.Equal(t, "30.00", parts[0].String())
	assert.Equal(t, "30.00", parts[1].String())
	assert.Equal(t, "30.00", parts[2].String())
}

func TestAllocateDistributesRemainderHalfEven(t *testing.T) {
	parts, err := mustMoney("100.00").Allocate(3)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	total := money.Sum(parts)
	assert.Equal(t, "100.00", total.String())
	assert.Equal(t, "33.34", parts[0].String())
	assert.Equal(t, "33.33", parts[1].String())
	assert.Equal(t, "33.33", parts[2].String())
}

func TestAllocateSinglePartReturnsSameValue(t *testing.T) {
	parts, err := mustMoney("77.77").Allocate(1)
	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Equal(t, "77.77", parts[0].String())
}

func TestAllocateRejectsNonPositiveParts(t *testing.T) {
	_, err := mustMoney("10.00").Allocate(0)
	assert.Error(t, err)
	_, err = mustMoney("10.00").Allocate(-1)
	assert.Error(t, err)
}

func TestIsNegativeReturnsTrueForNegativeAmount(t *testing.T) {
	assert.True(t, mustMoney("-1.00").IsNegative())
	assert.False(t, money.Zero().IsNegative())
	assert.False(t, mustMoney("0.01").IsNegative())
}

func TestSumOfEmptyListIsZero(t *testing.T) {
	assert.Equal(t, "0.00", money.Sum(nil).String())
	assert.Equal(t, "0.00", money.Sum([]money.Money{}).String())
}

func TestSumAccumulatesValues(t *testing.T) {
	total := money.Sum([]money.Money{
		mustMoney("10.10"),
		mustMoney("20.20"),
		mustMoney("-5.00"),
	})
	assert.Equal(t, "25.30", total.String())
}

func TestCentsConversion(t *testing.T) {
	m := mustMoney("123.45")
	assert.Equal(t, int64(12345), m.ToCents())

	from := money.FromCents(12345)
	assert.Equal(t, "123.45", from.String())
}

func TestMinAndMax(t *testing.T) {
	a := mustMoney("10.00")
	b := mustMoney("20.00")

	assert.Equal(t, "10.00", money.Min(a, b).String())
	assert.Equal(t, "20.00", money.Max(a, b).String())
}

func TestRatioAndPercent(t *testing.T) {
	a := mustMoney("25.00")
	base := mustMoney("100.00")

	assert.True(t, a.RatioOver(base).Equal(decimal.NewFromFloat(0.25)))
	assert.True(t, a.PercentOver(base).Equal(decimal.NewFromFloat(25.00)))
}

func TestJSONMarshaling(t *testing.T) {
	m := mustMoney("49.90")
	data, err := json.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, "49.90", string(data))

	var unmarshaled money.Money
	err = json.Unmarshal([]byte("49.90"), &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, "49.90", unmarshaled.String())
}
