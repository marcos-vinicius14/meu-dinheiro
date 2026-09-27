package engine

import (
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
)

// WhatIfSimulator simula o impacto de uma compra ao longo de N ciclos mensais sem persistência.
type WhatIfSimulator struct {
	engine *PredictiveEngine
}

func NewWhatIfSimulator(engine *PredictiveEngine) *WhatIfSimulator {
	return &WhatIfSimulator{engine: engine}
}

func (s *WhatIfSimulator) Simulate(
	baseContext CycleContext,
	baseState CurrentState,
	purchase SimulatedPurchase,
) ([]CycleSimulation, error) {
	if purchase.Installments <= 0 {
		purchase.Installments = 1
	}

	allocations, err := purchase.TotalAmount.Allocate(purchase.Installments)
	if err != nil {
		return nil, err
	}

	firstCycleStart := dateinterval.NormalizeDate(baseContext.CycleInterval.StartDate())
	purchaseStart := dateinterval.NormalizeDate(purchase.FirstDueDate)

	baselineResult := s.engine.Calculate(baseContext, baseState)
	baselineS2S := baselineResult.S2SToday

	simulations := make([]CycleSimulation, 0, purchase.Installments)

	for i := 0; i < purchase.Installments; i++ {
		// Avança i meses a partir do primeiro dia do primeiro ciclo
		firstOfMonth := time.Date(firstCycleStart.Year(), firstCycleStart.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

		cycleInterval, _ := dateinterval.New(firstOfMonth, lastOfMonth)

		var cycleCurrentDate time.Time
		if i == 0 {
			cycleCurrentDate = dateinterval.NormalizeDate(baseContext.CurrentDate)
		} else {
			cycleCurrentDate = cycleInterval.StartDate()
		}

		transactions := make([]TransactionSnapshot, len(baseState.Transactions))
		copy(transactions, baseState.Transactions)

		// Vencimento da parcela correspondente
		installmentDueDate := s.installmentDueDateInCycle(purchaseStart, firstOfMonth)
		if !installmentDueDate.IsZero() {
			transactions = append(transactions, TransactionSnapshot{
				Type:    TypeInstallmentExpense,
				Status:  StatusCommitted,
				Amount:  allocations[i],
				DueDate: installmentDueDate,
			})
		}

		effectiveState := CurrentState{
			LiquidBalance: baseState.LiquidBalance,
			Transactions:  transactions,
		}

		context := CycleContext{
			CycleInterval:     cycleInterval,
			CurrentDate:       cycleCurrentDate,
			TargetSavings:     baseContext.TargetSavings,
			FlexibleBudgetCap: baseContext.FlexibleBudgetCap,
		}

		result := s.engine.Calculate(context, effectiveState)

		reduction := baselineS2S.Subtract(result.S2SToday)
		var reductionPercent decimal.Decimal
		if !baselineS2S.IsZero() {
			reductionPercent = reduction.PercentOver(baselineS2S)
		}

		bottleneck := result.HealthStatus == HealthDeficitRisk || result.HealthStatus == HealthRestricted

		simulations = append(simulations, CycleSimulation{
			CycleInterval:       cycleInterval,
			EngineResult:        result,
			S2SReduction:        money.Max(reduction, money.Zero()),
			S2SReductionPercent: reductionPercent,
			HealthStatus:        result.HealthStatus,
			Bottleneck:          bottleneck,
		})
	}

	return simulations, nil
}

// Data da parcela no ciclo: mesmo day-of-month da primeira parcela, ajustado para o fim do mês se o dia não existir.
func (s *WhatIfSimulator) installmentDueDateInCycle(firstDueDate, cycleMonthStart time.Time) time.Time {
	monthsBetween := (cycleMonthStart.Year()-firstDueDate.Year())*12 + int(cycleMonthStart.Month()-firstDueDate.Month())
	if monthsBetween < 0 {
		return time.Time{}
	}

	// Adiciona os meses à primeira data de vencimento
	targetYear := cycleMonthStart.Year()
	targetMonth := cycleMonthStart.Month()
	targetDay := firstDueDate.Day()

	// Checa quantos dias o mês de destino tem
	lastDayOfMonth := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if targetDay > lastDayOfMonth {
		targetDay = lastDayOfMonth
	}

	return time.Date(targetYear, targetMonth, targetDay, 0, 0, 0, 0, time.UTC)
}
