package engine_test

import (
	"testing"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseContext() engine.CycleContext {
	return engine.CycleContext{
		CycleInterval:     mustInterval(cycleStart, cycleEnd),
		CurrentDate:       today,
		TargetSavings:     mustMoney("100.00"),
		FlexibleBudgetCap: mustMoney("900.00"),
	}
}

func baseState() engine.CurrentState {
	return engine.CurrentState{
		LiquidBalance: mustMoney("1000.00"),
		Transactions: []engine.TransactionSnapshot{
			{
				Type:    engine.TypeIncome,
				Status:  engine.StatusProjected,
				Amount:  mustMoney("500.00"),
				DueDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			},
			{
				Type:    engine.TypeFixedExpense,
				Status:  engine.StatusProjected,
				Amount:  mustMoney("300.00"),
				DueDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				Type:    engine.TypeInstallmentExpense,
				Status:  engine.StatusCommitted,
				Amount:  mustMoney("200.00"),
				DueDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
			},
		},
	}
}

func TestSinglePurchaseInCurrentCycleDegradesS2S(t *testing.T) {
	simulator := engine.NewWhatIfSimulator(engine.NewPredictiveEngine())
	result, err := simulator.Simulate(baseContext(), baseState(), engine.SimulatedPurchase{
		TotalAmount:  mustMoney("300.00"),
		Installments: 1,
		FirstDueDate: today,
	})
	require.NoError(t, err)
	require.Len(t, result, 1)

	first := result[0]
	assert.Equal(t, cycleStart, first.CycleInterval.StartDate())
	assert.Equal(t, "33.33", first.EngineResult.S2SToday.String())
	assert.Equal(t, "16.67", first.S2SReduction.String())
	assert.Equal(t, "33.34", first.S2SReductionPercent.String())
	assert.Equal(t, engine.HealthRestricted, first.HealthStatus)
	assert.True(t, first.Bottleneck)
}

func TestInstallmentPurchaseProjectsAcrossSixCycles(t *testing.T) {
	simulator := engine.NewWhatIfSimulator(engine.NewPredictiveEngine())
	result, err := simulator.Simulate(baseContext(), baseState(), engine.SimulatedPurchase{
		TotalAmount:  mustMoney("600.00"),
		Installments: 6,
		FirstDueDate: today,
	})
	require.NoError(t, err)
	require.Len(t, result, 6)

	assert.Equal(t, cycleStart, result[0].CycleInterval.StartDate())
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), result[1].CycleInterval.StartDate())
	assert.Equal(t, time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), result[5].CycleInterval.StartDate())

	assert.Equal(t, "44.44", result[0].EngineResult.S2SToday.String())

	expectedByCycle := []string{"25.80", "26.66", "25.80", "25.80", "28.57"}
	for i := 1; i < 6; i++ {
		assert.Equal(t, expectedByCycle[i-1], result[i].EngineResult.S2SToday.String())
	}
}

func TestTwelveMonthSimulationReportsBottleneckWhenLiquidityRunsOut(t *testing.T) {
	simulator := engine.NewWhatIfSimulator(engine.NewPredictiveEngine())
	result, err := simulator.Simulate(baseContext(), baseState(), engine.SimulatedPurchase{
		TotalAmount:  mustMoney("24000.00"),
		Installments: 12,
		FirstDueDate: today,
	})
	require.NoError(t, err)
	require.Len(t, result, 12)

	for _, cycle := range result {
		assert.Equal(t, engine.HealthDeficitRisk, cycle.HealthStatus)
		assert.True(t, cycle.Bottleneck)
	}
}

func TestBottleneckIdentifiesCyclesThatDegradeHealth(t *testing.T) {
	simulator := engine.NewWhatIfSimulator(engine.NewPredictiveEngine())
	result, err := simulator.Simulate(baseContext(), baseState(), engine.SimulatedPurchase{
		TotalAmount:  mustMoney("12000.00"),
		Installments: 12,
		FirstDueDate: today,
	})
	require.NoError(t, err)
	require.Len(t, result, 12)

	assert.Equal(t, engine.HealthDeficitRisk, result[0].HealthStatus)
	assert.True(t, result[0].Bottleneck)

	for i := 1; i < 12; i++ {
		assert.Equal(t, engine.HealthDeficitRisk, result[i].HealthStatus)
		assert.True(t, result[i].Bottleneck)
	}
}

func TestNoImpactWhenPurchaseIsZero(t *testing.T) {
	simulator := engine.NewWhatIfSimulator(engine.NewPredictiveEngine())
	result, err := simulator.Simulate(baseContext(), baseState(), engine.SimulatedPurchase{
		TotalAmount:  mustMoney("0.00"),
		Installments: 1,
		FirstDueDate: today,
	})
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "0.00", result[0].S2SReduction.String())
}
