package dateinterval_test

import (
	"testing"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestCreatesValidInterval(t *testing.T) {
	start := date(2026, 9, 1)
	end := date(2026, 9, 30)

	interval, err := dateinterval.New(start, end)
	require.NoError(t, err)

	assert.Equal(t, start, interval.StartDate())
	assert.Equal(t, end, interval.EndDate())
}

func TestRejectsStartAfterEnd(t *testing.T) {
	_, err := dateinterval.New(date(2026, 9, 30), date(2026, 9, 1))
	assert.Error(t, err)
}

func TestAcceptsSingleDayInterval(t *testing.T) {
	d := date(2026, 9, 15)
	interval, err := dateinterval.New(d, d)
	require.NoError(t, err)

	assert.Equal(t, 1, interval.DaysRemaining(d))
}

func TestDaysRemainingIsInclusiveOfCurrentDay(t *testing.T) {
	interval, err := dateinterval.New(date(2026, 9, 1), date(2026, 9, 30))
	require.NoError(t, err)

	assert.Equal(t, 18, interval.DaysRemaining(date(2026, 9, 13)))
}

func TestDaysRemainingNeverBelowOne(t *testing.T) {
	interval, err := dateinterval.New(date(2026, 9, 1), date(2026, 9, 30))
	require.NoError(t, err)

	assert.Equal(t, 1, interval.DaysRemaining(date(2026, 10, 5)))
}

func TestDaysRemainingOnLastDayIsOne(t *testing.T) {
	interval, err := dateinterval.New(date(2026, 9, 1), date(2026, 9, 30))
	require.NoError(t, err)

	assert.Equal(t, 1, interval.DaysRemaining(date(2026, 9, 30)))
}

func TestContains(t *testing.T) {
	interval, err := dateinterval.New(date(2026, 9, 1), date(2026, 9, 30))
	require.NoError(t, err)

	assert.True(t, interval.Contains(date(2026, 9, 1)))
	assert.True(t, interval.Contains(date(2026, 9, 15)))
	assert.True(t, interval.Contains(date(2026, 9, 30)))
	assert.False(t, interval.Contains(date(2026, 8, 31)))
	assert.False(t, interval.Contains(date(2026, 10, 1)))
}

func TestMonthOfMatchesCivilMonth(t *testing.T) {
	interval := dateinterval.MonthOf(date(2026, 9, 15))
	assert.Equal(t, date(2026, 9, 1), interval.StartDate())
	assert.Equal(t, date(2026, 9, 30), interval.EndDate())
}

func TestCycleOfDefaultFirstDayMatchesCivilMonth(t *testing.T) {
	interval := dateinterval.CycleOf(date(2026, 9, 15), 1)
	assert.Equal(t, date(2026, 9, 1), interval.StartDate())
	assert.Equal(t, date(2026, 9, 30), interval.EndDate())
}

func TestCycleOfCustomDaySameMonth(t *testing.T) {
	// Hoje é 15/09 e o ciclo começa no dia 05
	interval := dateinterval.CycleOf(date(2026, 9, 15), 5)
	assert.Equal(t, date(2026, 9, 5), interval.StartDate())
	assert.Equal(t, date(2026, 10, 4), interval.EndDate())
	assert.True(t, interval.Contains(date(2026, 9, 15)))
}

func TestCycleOfCustomDayPreviousMonth(t *testing.T) {
	// Hoje é 02/09 e o ciclo começa no dia 05 (então começou em 05/08 e vai até 04/09)
	interval := dateinterval.CycleOf(date(2026, 9, 2), 5)
	assert.Equal(t, date(2026, 8, 5), interval.StartDate())
	assert.Equal(t, date(2026, 9, 4), interval.EndDate())
	assert.True(t, interval.Contains(date(2026, 9, 2)))
}

func TestCycleOfYearTransition(t *testing.T) {
	// Hoje é 02/01/2027 e o ciclo começa no dia 05 (começou em 05/12/2026 e vai até 04/01/2027)
	interval := dateinterval.CycleOf(date(2027, 1, 2), 5)
	assert.Equal(t, date(2026, 12, 5), interval.StartDate())
	assert.Equal(t, date(2027, 1, 4), interval.EndDate())
	assert.True(t, interval.Contains(date(2027, 1, 2)))
}

func TestCycleOfClampsInvalidDays(t *testing.T) {
	// Valores < 1 ou > 28 devem sofrer fallback para dia 1
	interval0 := dateinterval.CycleOf(date(2026, 9, 15), 0)
	assert.Equal(t, date(2026, 9, 1), interval0.StartDate())

	interval31 := dateinterval.CycleOf(date(2026, 9, 15), 31)
	assert.Equal(t, date(2026, 9, 1), interval31.StartDate())
}
