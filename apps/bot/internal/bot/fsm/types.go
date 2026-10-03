package fsm

import "time"

// State representa o estado atual do usuário na Máquina de Estados Finitos (FSM).
type State string

const (
	StateIdle                       State = "IDLE"
	StateWaitingBalance             State = "WAITING_BALANCE"
	StateWaitingCycleDay            State = "WAITING_CYCLE_DAY"
	StateWaitingFixedExpenses       State = "WAITING_FIXED_EXPENSES"
	StateWaitingEmergencyFundChoice State = "WAITING_EMERGENCY_FUND_CHOICE"
	StateWaitingSavingsTarget       State = "WAITING_SAVINGS_TARGET"
	StateWaitingInvestmentChoice    State = "WAITING_INVESTMENT_CHOICE"
	StateWaitingInvestmentInput     State = "WAITING_INVESTMENT_INPUT"
)

// Action define as ações tipadas disparadas por botões inline (callbacks) da FSM.
type Action string

const (
	ActionFinishFixedExpenses Action = "finish_fixed_expenses"
	ActionFund6               Action = "fund_6"
	ActionFund12              Action = "fund_12"
	ActionSkipInvestments     Action = "skip_investments"
	ActionAddInvestment       Action = "add_investment"
	ActionFinishInvestments   Action = "finish_investments"
)

// FixedExpenseData armazena temporariamente cada despesa essencial cadastrada.
type FixedExpenseData struct {
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	CategoryName string  `json:"category_name"`
}

// InvestmentData armazena temporariamente cada ativo de investimento adicionado.
type InvestmentData struct {
	Ticker       string  `json:"ticker"`
	Quantity     float64 `json:"quantity"`
	AveragePrice float64 `json:"average_price"`
}

// Session representa o estado completo de conversação e dados acumulados de um usuário.
type Session struct {
	TelegramID           int64              `json:"telegram_id"`
	ChatID               int64              `json:"chat_id"`
	FirstName            string             `json:"first_name"`
	Username             string             `json:"username"`
	CurrentState         State              `json:"current_state"`
	InitialBalance       float64            `json:"initial_balance"`
	CycleStartDay        int                `json:"cycle_start_day"`
	FixedExpenses        []FixedExpenseData `json:"fixed_expenses"`
	EmergencyFundMonths  int                `json:"emergency_fund_months"`
	MonthlyEssentialCost float64            `json:"monthly_essential_cost"`
	TargetSavings        float64            `json:"target_savings"`
	Investments          []InvestmentData   `json:"investments"`
	LastActiveAt         time.Time          `json:"last_active_at"`
}
