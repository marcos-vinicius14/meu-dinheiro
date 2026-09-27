package engine_test

import (
	"testing"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/stretchr/testify/assert"
)

var (
	cycleStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	cycleEnd   = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	today      = time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
)

func mustMoney(s string) money.Money {
	m, err := money.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return m
}

func mustInterval(start, end time.Time) dateinterval.DateInterval {
	iv, err := dateinterval.New(start, end)
	if err != nil {
		panic(err)
	}
	return iv
}

func context(targetSavings, flexibleBudgetCap money.Money) engine.CycleContext {
	return engine.CycleContext{
		CycleInterval:     mustInterval(cycleStart, cycleEnd),
		CurrentDate:       today,
		TargetSavings:     targetSavings,
		FlexibleBudgetCap: flexibleBudgetCap,
	}
}

func confirmedFlexible(amount money.Money, d time.Time) engine.TransactionSnapshot {
	return engine.TransactionSnapshot{
		Type:    engine.TypeFlexibleExpense,
		Status:  engine.StatusConfirmed,
		Amount:  amount,
		DueDate: d,
	}
}

func projected(txType engine.TransactionType, amount money.Money, dueDate time.Time) engine.TransactionSnapshot {
	return engine.TransactionSnapshot{
		Type:    txType,
		Status:  engine.StatusProjected,
		Amount:  amount,
		DueDate: dueDate,
	}
}

func committed(txType engine.TransactionType, amount money.Money, dueDate time.Time) engine.TransactionSnapshot {
	return engine.TransactionSnapshot{
		Type:    txType,
		Status:  engine.StatusCommitted,
		Amount:  amount,
		DueDate: dueDate,
	}
}

func TestNoFlexibleSpendingSoFarGivesFullDailyAllowance(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("100.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions: []engine.TransactionSnapshot{
				projected(engine.TypeIncome, mustMoney("500.00"), time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)),
				projected(engine.TypeFixedExpense, mustMoney("300.00"), time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)),
				committed(engine.TypeInstallmentExpense, mustMoney("200.00"), time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "50.00", result.S2SToday.String())
	assert.Equal(t, "900.00", result.RemainingFlexible.String())
	assert.Equal(t, "0.00", result.ProjectedBalance.String())
	assert.Equal(t, engine.HealthHealthy, result.HealthStatus)
	assert.Equal(t, 18, result.DaysRemaining)
}

func TestOverspendYesterdayRecalibratesDown(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions: []engine.TransactionSnapshot{
				confirmedFlexible(mustMoney("800.00"), time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "100.00", result.RemainingFlexible.String())
	assert.Equal(t, "5.55", result.S2SToday.String())
	assert.Equal(t, engine.HealthHealthy, result.HealthStatus)
}

func TestUnderspendYesterdayRecalibratesUp(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("500.00"),
			Transactions: []engine.TransactionSnapshot{
				confirmedFlexible(mustMoney("100.00"), time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "500.00", result.RemainingFlexible.String())
	assert.Equal(t, "27.77", result.S2SToday.String())
}

func TestProjectedDeficitFromCommitmentsForcesZeroS2SAndDeficitRisk(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("300.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("100.00"),
			Transactions: []engine.TransactionSnapshot{
				projected(engine.TypeIncome, mustMoney("200.00"), time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)),
				projected(engine.TypeFixedExpense, mustMoney("500.00"), time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)),
				committed(engine.TypeInstallmentExpense, mustMoney("400.00"), time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "0.00", result.S2SToday.String())
	assert.Equal(t, engine.HealthDeficitRisk, result.HealthStatus)
	assert.Equal(t, "0.00", result.RemainingFlexible.String())
}

func TestNegativeProjectedBalanceButPositiveFlexCapacityIsRestricted(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("300.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("400.00"),
			Transactions: []engine.TransactionSnapshot{
				projected(engine.TypeFixedExpense, mustMoney("200.00"), time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "11.11", result.S2SToday.String())
	assert.Equal(t, engine.HealthRestricted, result.HealthStatus)
	assert.Equal(t, "-100.00", result.ProjectedBalance.String())
}

func TestLastDayOfCycleHasSingleDayRemaining(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	lastDayContext := engine.CycleContext{
		CycleInterval:     mustInterval(cycleStart, cycleEnd),
		CurrentDate:       cycleEnd,
		TargetSavings:     mustMoney("0.00"),
		FlexibleBudgetCap: mustMoney("300.00"),
	}

	result := eng.Calculate(lastDayContext, engine.CurrentState{
		LiquidBalance: mustMoney("300.00"),
		Transactions:  nil,
	})

	assert.Equal(t, 1, result.DaysRemaining)
	assert.Equal(t, "300.00", result.S2SToday.String())
}

func TestConfirmedIncomeBeforeTodayCountsTowardLiquidBalanceNotProjectedIncome(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions: []engine.TransactionSnapshot{
				{
					Type:    engine.TypeIncome,
					Status:  engine.StatusConfirmed,
					Amount:  mustMoney("500.00"),
					DueDate: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	)

	assert.Equal(t, "900.00", result.RemainingFlexible.String())
}

func TestCanceledTransactionsAreIgnored(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions: []engine.TransactionSnapshot{
				{
					Type:    engine.TypeFixedExpense,
					Status:  engine.StatusCanceled,
					Amount:  mustMoney("10000.00"),
					DueDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	)

	assert.Equal(t, "900.00", result.RemainingFlexible.String())
	assert.Equal(t, engine.HealthHealthy, result.HealthStatus)
}

func TestFlexibleSpendingOutsideCycleDoesNotConsumeBudget(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("0.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions: []engine.TransactionSnapshot{
				confirmedFlexible(mustMoney("500.00"), time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)),
			},
		},
	)

	assert.Equal(t, "900.00", result.RemainingFlexible.String())
}

func TestTargetSavingsIsReserveBeforeFlexibleBudget(t *testing.T) {
	eng := engine.NewPredictiveEngine()
	result := eng.Calculate(
		context(mustMoney("400.00"), mustMoney("900.00")),
		engine.CurrentState{
			LiquidBalance: mustMoney("1000.00"),
			Transactions:  nil,
		},
	)

	assert.Equal(t, "600.00", result.RemainingFlexible.String())
	assert.Equal(t, engine.HealthRestricted, result.HealthStatus)
}
