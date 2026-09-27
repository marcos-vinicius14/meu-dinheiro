package engine

import (
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
)

// PredictiveEngine realiza os cálculos do Safe-To-Spend diário e saúde do ciclo.
type PredictiveEngine struct{}

func NewPredictiveEngine() *PredictiveEngine {
	return &PredictiveEngine{}
}

func (e *PredictiveEngine) Calculate(context CycleContext, state CurrentState) EngineResult {
	interval := context.CycleInterval
	today := dateinterval.NormalizeDate(context.CurrentDate)

	daysRemaining := interval.DaysRemaining(today)

	flexSpent := e.sumUntilYesterday(state.Transactions, today, interval)
	projectedIncome := e.sumForward(state.Transactions, today, interval, TypeIncome)
	fixedExpenses := e.sumForward(state.Transactions, today, interval, TypeFixedExpense)
	committedExpenses := e.sumForwardCommittedInstallments(state.Transactions, today, interval)

	remainingBudget := money.Max(context.FlexibleBudgetCap.Subtract(flexSpent), money.Zero())

	netLiquidityBeforeFlex := state.LiquidBalance.
		Add(projectedIncome).
		Subtract(fixedExpenses).
		Subtract(committedExpenses).
		Subtract(context.TargetSavings)

	remainingFlexible := money.Min(remainingBudget, netLiquidityBeforeFlex)
	projectedBalance := netLiquidityBeforeFlex.Subtract(remainingBudget)

	var s2sToday money.Money
	var healthStatus HealthStatus

	if remainingFlexible.IsNegative() || remainingFlexible.IsZero() {
		s2sToday = money.Zero()
		healthStatus = HealthDeficitRisk
	} else {
		s2sToday = e.floorPerDay(remainingFlexible, daysRemaining)
		if projectedBalance.IsNegative() {
			healthStatus = HealthRestricted
		} else {
			healthStatus = HealthHealthy
		}
	}

	return EngineResult{
		S2SToday:               s2sToday,
		RemainingFlexible:      money.Max(remainingFlexible, money.Zero()),
		ProjectedBalance:       projectedBalance,
		NetLiquidityBeforeFlex: netLiquidityBeforeFlex,
		FlexSpent:              flexSpent,
		ProjectedIncome:        projectedIncome,
		CommittedExpenses:      fixedExpenses.Add(committedExpenses),
		HealthStatus:           healthStatus,
		DaysRemaining:          daysRemaining,
	}
}

// Despesas flexíveis CONFIRMADAS entre o início do ciclo e o dia anterior ao atual.
func (e *PredictiveEngine) sumUntilYesterday(
	transactions []TransactionSnapshot,
	today time.Time,
	interval dateinterval.DateInterval,
) money.Money {
	var total money.Money
	for _, t := range transactions {
		dueDate := dateinterval.NormalizeDate(t.DueDate)
		if t.Type == TypeFlexibleExpense &&
			t.Status == StatusConfirmed &&
			!dueDate.Before(interval.StartDate()) &&
			dueDate.Before(today) {
			total = total.Add(t.Amount)
		}
	}
	return total
}

// Transações PROJECTED/COMMITTED do tipo informado entre hoje e o fim do ciclo.
func (e *PredictiveEngine) sumForward(
	transactions []TransactionSnapshot,
	today time.Time,
	interval dateinterval.DateInterval,
	txType TransactionType,
) money.Money {
	var total money.Money
	for _, t := range transactions {
		dueDate := dateinterval.NormalizeDate(t.DueDate)
		if t.Type == txType &&
			(t.Status == StatusProjected || t.Status == StatusCommitted) &&
			!dueDate.Before(today) &&
			!dueDate.After(interval.EndDate()) {
			total = total.Add(t.Amount)
		}
	}
	return total
}

// Parcelas COMMITTED de compras parceladas entre hoje e o fim do ciclo.
func (e *PredictiveEngine) sumForwardCommittedInstallments(
	transactions []TransactionSnapshot,
	today time.Time,
	interval dateinterval.DateInterval,
) money.Money {
	var total money.Money
	for _, t := range transactions {
		dueDate := dateinterval.NormalizeDate(t.DueDate)
		if t.Type == TypeInstallmentExpense &&
			t.Status == StatusCommitted &&
			!dueDate.Before(today) &&
			!dueDate.After(interval.EndDate()) {
			total = total.Add(t.Amount)
		}
	}
	return total
}

// Divisão diária com arredondamento para baixo em centavos (conservador).
func (e *PredictiveEngine) floorPerDay(total money.Money, days int) money.Money {
	if days <= 0 {
		return money.Zero()
	}
	cents := total.ToCents()
	dailyCents := cents / int64(days)
	return money.FromCents(dailyCents)
}
