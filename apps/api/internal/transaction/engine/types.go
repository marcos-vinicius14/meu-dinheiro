package engine

import (
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TypeIncome             TransactionType = "INCOME"
	TypeFixedExpense       TransactionType = "FIXED_EXPENSE"
	TypeFlexibleExpense    TransactionType = "FLEXIBLE_EXPENSE"
	TypeInstallmentExpense TransactionType = "INSTALLMENT_EXPENSE"
)

type TransactionStatus string

const (
	StatusProjected TransactionStatus = "PROJECTED"
	StatusCommitted TransactionStatus = "COMMITTED"
	StatusConfirmed TransactionStatus = "CONFIRMED"
	StatusCanceled  TransactionStatus = "CANCELED"
)

type HealthStatus string

const (
	HealthHealthy     HealthStatus = "HEALTHY"
	HealthRestricted  HealthStatus = "RESTRICTED"
	HealthDeficitRisk HealthStatus = "DEFICIT_RISK"
)

type TransactionSnapshot struct {
	Type    TransactionType
	Status  TransactionStatus
	Amount  money.Money
	DueDate time.Time
}

type CycleContext struct {
	CycleInterval     dateinterval.DateInterval
	CurrentDate       time.Time
	TargetSavings     money.Money
	FlexibleBudgetCap money.Money
}

type CurrentState struct {
	LiquidBalance money.Money
	Transactions  []TransactionSnapshot
}

type EngineResult struct {
	S2SToday               money.Money
	RemainingFlexible      money.Money
	ProjectedBalance       money.Money
	NetLiquidityBeforeFlex money.Money
	FlexSpent              money.Money
	ProjectedIncome        money.Money
	CommittedExpenses      money.Money
	HealthStatus           HealthStatus
	DaysRemaining          int
}

type SimulatedPurchase struct {
	TotalAmount  money.Money
	Installments int
	FirstDueDate time.Time
}

type CycleSimulation struct {
	CycleInterval       dateinterval.DateInterval
	EngineResult        EngineResult
	S2SReduction        money.Money
	S2SReductionPercent decimal.Decimal
	HealthStatus        HealthStatus
	Bottleneck          bool
}
