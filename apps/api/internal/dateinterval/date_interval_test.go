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
