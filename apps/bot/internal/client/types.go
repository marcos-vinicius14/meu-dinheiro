package client

import (
	"encoding/json"
	"strconv"
	"strings"
)

// FlexFloat permite deserializar números tanto como float64 quanto como string numérica vinda do JSON.
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), "\"")
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = FlexFloat(val)
	return nil
}

func (f FlexFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}

// --- AUTENTICAÇÃO ---

type AuthorizeChallengeRequest struct {
	Token      string `json:"token"`
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
}

// --- ONBOARDING & SETUP INICIAL ---

type FixedExpenseInput struct {
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	CategoryName string  `json:"category_name"`
}

type InitialInvestmentInput struct {
	Ticker       string  `json:"ticker"`
	Quantity     float64 `json:"quantity"`
	AveragePrice float64 `json:"average_price"`
}

type OnboardingRequest struct {
	TelegramID           int64                    `json:"telegram_id"`
	FirstName            string                   `json:"first_name"`
	Username             *string                  `json:"username,omitempty"`
	InitialBalance       float64                  `json:"initial_balance"`
	CycleStartDay        int                      `json:"cycle_start_day"`
	FixedExpenses        []FixedExpenseInput      `json:"fixed_expenses"`
	EmergencyFundMonths  int                      `json:"emergency_fund_months"`
	CurrentEmergencyFund float64                  `json:"current_emergency_fund,omitempty"`
	TargetSavings        float64                  `json:"target_savings"`
	FlexibleBudgetCap    float64                  `json:"flexible_budget_cap"`
	Investments          []InitialInvestmentInput `json:"investments"`
}

type CycleResponse struct {
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	DaysRemaining int     `json:"days_remaining"`
	S2SToday      float64 `json:"s2s_today"`
	HealthStatus  string  `json:"health_status"`
}

type EmergencyFundResponse struct {
	MonthlyEssentialCost float64   `json:"monthly_essential_cost"`
	Suggested6x          float64   `json:"suggested_6x"`
	Suggested12x         float64   `json:"suggested_12x"`
	ChosenTarget         float64   `json:"chosen_target"`
	ChosenMonths         int       `json:"chosen_months"`
	CurrentBalance       float64   `json:"current_balance,omitempty"`
	MonthsCovered        float64   `json:"months_covered"`
	ProgressPercent      FlexFloat `json:"progress_percent"`
}

type OnboardingResponse struct {
	Message            string                `json:"message"`
	UserID             string                `json:"user_id"`
	Cycle              CycleResponse         `json:"cycle"`
	EmergencyFund      EmergencyFundResponse `json:"emergency_fund"`
	TotalLiquidBalance float64               `json:"total_liquid_balance"`
	TotalInvested      float64               `json:"total_invested"`
	TotalNetWorth      float64               `json:"total_net_worth"`
}

// --- CONTEXTO FINANCEIRO CONSOLIDADO ---

type UserFinancialContext struct {
	User struct {
		ID                  string  `json:"id"`
		TelegramID          int64   `json:"telegram_id"`
		FirstName           string  `json:"first_name"`
		Username            *string `json:"username,omitempty"`
		TargetSavings       float64 `json:"target_savings"`
		FlexibleBudgetCap   float64 `json:"flexible_budget_cap"`
		EmergencyFundTarget float64 `json:"emergency_fund_target"`
		EmergencyFundMonths int     `json:"emergency_fund_months"`
		CycleStartDay       int     `json:"cycle_start_day"`
	} `json:"user"`
	Accounts []struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		CurrentBalance float64 `json:"current_balance"`
	} `json:"accounts"`
	TotalLiquidBalance float64 `json:"total_liquid_balance"`
	Cycle              struct {
		StartDate         string  `json:"start_date"`
		EndDate           string  `json:"end_date"`
		DaysRemaining     int     `json:"days_remaining"`
		S2SToday          float64 `json:"s2s_today"`
		HealthStatus      string  `json:"health_status"`
		ProjectedBalance  float64 `json:"projected_balance"`
		RemainingFlexible float64 `json:"remaining_flexible"`
		FlexibleSpent     float64 `json:"flexible_spent"`
	} `json:"cycle"`
	EmergencyFund struct {
		MonthlyEssentialCost float64   `json:"monthly_essential_cost"`
		Target               float64   `json:"target"`
		Months               int       `json:"months"`
		MonthsCovered        float64   `json:"months_covered"`
		ProgressPercent      FlexFloat `json:"progress_percent"`
	} `json:"emergency_fund"`
	Investments []struct {
		ID           string    `json:"id"`
		Ticker       string    `json:"ticker"`
		Quantity     FlexFloat `json:"quantity"`
		AveragePrice float64   `json:"average_price"`
		TotalCost    float64   `json:"total_cost"`
	} `json:"investments"`
	TotalInvested float64 `json:"total_invested"`
	TotalNetWorth float64 `json:"total_net_worth"`
}

// --- INVESTIMENTOS ---

type AddInvestmentRequest struct {
	TelegramID int64   `json:"telegram_id"`
	Ticker     string  `json:"ticker"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
}

type AddInvestmentResponse struct {
	Investment struct {
		ID           string    `json:"id"`
		Ticker       string    `json:"ticker"`
		Quantity     FlexFloat `json:"quantity"`
		AveragePrice float64   `json:"average_price"`
		TotalCost    float64   `json:"total_cost"`
	} `json:"investment"`
	TotalInvested float64 `json:"total_invested"`
	TotalNetWorth float64 `json:"total_net_worth"`
}

// --- GASTOS RÁPIDOS ---

type QuickExpenseRequest struct {
	TelegramID   int64   `json:"telegram_id"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	CategoryName string  `json:"category_name,omitempty"`
	Date         string  `json:"date,omitempty"`
}

type QuickExpenseResponse struct {
	Transaction struct {
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	} `json:"transaction"`
	CategoryName  string  `json:"category_name"`
	PreviousS2S   float64 `json:"previous_s2s"`
	NewS2S        float64 `json:"new_s2s"`
	HealthStatus  string  `json:"health_status"`
	DaysRemaining int     `json:"days_remaining"`
}

// --- SIMULAÇÃO WHAT-IF (PRED-03, D-05..D-08) ---

type SimulationRequest struct {
	TelegramID   int64   `json:"telegram_id"`
	Amount       float64 `json:"amount"`
	Installments int     `json:"installments,omitempty"`
	FirstDueDate string  `json:"first_due_date,omitempty"`
}

type SimulationCycleItem struct {
	CycleStart          string    `json:"cycle_start"`
	CycleEnd            string    `json:"cycle_end"`
	S2SToday            float64   `json:"s2s_today"`
	S2SReduction        float64   `json:"s2s_reduction"`
	S2SReductionPercent FlexFloat `json:"s2s_reduction_percent"`
	ProjectedBalance    float64   `json:"projected_balance"`
	HealthStatus        string    `json:"health_status"`
	Bottleneck          bool      `json:"bottleneck"`
}

type SimulationResponse struct {
	TotalAmount                  float64               `json:"total_amount"`
	Installments                 int                   `json:"installments"`
	InstallmentAmount            float64               `json:"installment_amount"`
	CurrentCycleS2SBefore        float64               `json:"current_cycle_s2s_before"`
	CurrentCycleS2SAfter         float64               `json:"current_cycle_s2s_after"`
	CurrentCycleReduction        float64               `json:"current_cycle_reduction"`
	CurrentCycleReductionPercent FlexFloat             `json:"current_cycle_reduction_percent"`
	CriticalCycleNumber          int                   `json:"critical_cycle_number"`
	CriticalCycleS2S             float64               `json:"critical_cycle_s2s"`
	CriticalCycleHealthStatus    string                `json:"critical_cycle_health_status"`
	DeficitRiskAlert             bool                  `json:"deficit_risk_alert"`
	RecommendationMessage        string                `json:"recommendation_message"`
	Cycles                       []SimulationCycleItem `json:"cycles"`
}

// --- CHECK-IN DIÁRIO (PRED-04, D-09, D-11, D-12) ---

type CheckInExpenseInput struct {
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	CategoryName string  `json:"category_name,omitempty"`
}

type DailyCheckInRequest struct {
	TelegramID        int64                 `json:"telegram_id"`
	Date              string                `json:"date,omitempty"`
	UntrackedExpenses []CheckInExpenseInput `json:"untracked_expenses,omitempty"`
}

type DailyCheckInResponse struct {
	S2SCalculated float64 `json:"s2s_calculated"`
	SpentToday    float64 `json:"spent_today"`
	DailyQuota    float64 `json:"daily_quota"`
	DeltaSavings  float64 `json:"delta_savings"`
	HealthStatus  string  `json:"health_status"`
	NextDayS2S    float64 `json:"next_day_s2s"`
	DaysRemaining int     `json:"days_remaining"`
	Message       string  `json:"message"`
}

// --- AJUSTE DE SALDO BANCÁRIO (D-10) ---

type AdjustBalanceRequest struct {
	TelegramID int64   `json:"telegram_id"`
	AccountID  *string `json:"account_id,omitempty"`
	NewBalance float64 `json:"new_balance"`
}

type AdjustBalanceResponse struct {
	AccountID          string  `json:"account_id"`
	AccountName        string  `json:"account_name"`
	PreviousBalance    float64 `json:"previous_balance"`
	NewBalance         float64 `json:"new_balance"`
	TotalLiquidBalance float64 `json:"total_liquid_balance"`
	Message            string  `json:"message"`
}

// --- ENTRADA DE RENDA / RECEITA ---

type IncomeRequest struct {
	TelegramID   int64   `json:"telegram_id"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	CategoryName string  `json:"category_name,omitempty"`
	Date         string  `json:"date,omitempty"`
}

type IncomeResponse struct {
	Transaction struct {
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	} `json:"transaction"`
	CategoryName       string  `json:"category_name"`
	PreviousS2S        float64 `json:"previous_s2s"`
	NewS2S             float64 `json:"new_s2s"`
	HealthStatus       string  `json:"health_status"`
	DaysRemaining      int     `json:"days_remaining"`
	TotalLiquidBalance float64 `json:"total_liquid_balance"`
}
