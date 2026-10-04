package botapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
)

type FixedExpenseInput struct {
	Description  string      `json:"description"`
	Amount       money.Money `json:"amount"`
	CategoryName string      `json:"category_name"`
}

type InitialInvestmentInput struct {
	Ticker       string          `json:"ticker"`
	Quantity     decimal.Decimal `json:"quantity"`
	AveragePrice money.Money     `json:"average_price"`
}

type OnboardingRequest struct {
	TelegramID           int64                    `json:"telegram_id"`
	FirstName            string                   `json:"first_name"`
	Username             *string                  `json:"username,omitempty"`
	InitialBalance       money.Money              `json:"initial_balance"`
	CycleStartDay        int                      `json:"cycle_start_day"` // 1 a 28, default 1
	FixedExpenses        []FixedExpenseInput      `json:"fixed_expenses"`
	EmergencyFundMonths  int                      `json:"emergency_fund_months"` // 6 ou 12, default 6
	CurrentEmergencyFund money.Money              `json:"current_emergency_fund"`
	TargetSavings        money.Money              `json:"target_savings"`
	FlexibleBudgetCap    money.Money              `json:"flexible_budget_cap"`
	Investments          []InitialInvestmentInput `json:"investments"`
}

type EmergencyFundResponse struct {
	MonthlyEssentialCost money.Money     `json:"monthly_essential_cost"`
	Suggested6x          money.Money     `json:"suggested_6x"`
	Suggested12x         money.Money     `json:"suggested_12x"`
	ChosenTarget         money.Money     `json:"chosen_target"`
	ChosenMonths         int             `json:"chosen_months"`
	CurrentBalance       money.Money     `json:"current_balance"`
	MonthsCovered        float64         `json:"months_covered"`
	ProgressPercent      decimal.Decimal `json:"progress_percent"`
}

type CycleResponse struct {
	StartDate     string              `json:"start_date"`
	EndDate       string              `json:"end_date"`
	DaysRemaining int                 `json:"days_remaining"`
	S2SToday      money.Money         `json:"s2s_today"`
	HealthStatus  engine.HealthStatus `json:"health_status"`
}

type OnboardingResponse struct {
	Message            string                `json:"message"`
	UserID             uuid.UUID             `json:"user_id"`
	Cycle              CycleResponse         `json:"cycle"`
	EmergencyFund      EmergencyFundResponse `json:"emergency_fund"`
	TotalLiquidBalance money.Money           `json:"total_liquid_balance"`
	TotalInvested      money.Money           `json:"total_invested"`
	TotalNetWorth      money.Money           `json:"total_net_worth"`
}

func (h *Handler) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	var req OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 || strings.TrimSpace(req.FirstName) == "" {
		web.Error(w, http.StatusBadRequest, "telegram_id e first_name são obrigatórios")
		return
	}

	ctx := r.Context()

	// 1. Localiza ou cria o usuário
	u, err := h.userRepo.FindOrCreateByTelegram(ctx, req.TelegramID, req.Username, req.FirstName)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao provisionar usuário")
		return
	}

	// 2. Atualiza saldo da conta bancária padrão
	accounts, err := h.bankAccountRepo.ListByUserID(ctx, u.ID)
	if err == nil && len(accounts) > 0 {
		_, _ = h.bankAccountRepo.Update(ctx, accounts[0].ID, u.ID, accounts[0].Name, req.InitialBalance)
	}

	// 2.1 Se informou valor inicial de reserva de emergência, provisiona conta Poupança/Reserva
	if req.CurrentEmergencyFund.IsPositive() {
		var savingsFound bool
		for _, acc := range accounts {
			if strings.EqualFold(acc.Name, "Reserva de Emergência") || acc.Type == "SAVINGS" {
				_, _ = h.bankAccountRepo.Update(ctx, acc.ID, u.ID, acc.Name, req.CurrentEmergencyFund)
				savingsFound = true
				break
			}
		}
		if !savingsFound && len(accounts) < 3 {
			_, _ = h.bankAccountRepo.Create(ctx, u.ID, "Reserva de Emergência", "SAVINGS", req.CurrentEmergencyFund)
		}
	}

	// 3. Processa e persiste despesas fixas essenciais
	categories, err := h.categoryRepo.ListByUserID(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao listar categorias")
		return
	}

	monthlyEssentialCost := money.Zero()
	now := time.Now().UTC()

	for _, exp := range req.FixedExpenses {
		if exp.Amount.IsPositive() {
			catID := matchCategory(categories, exp.CategoryName)
			tx := &transaction.Transaction{
				UserID:      u.ID,
				CategoryID:  catID,
				Description: exp.Description,
				Amount:      exp.Amount,
				Type:        engine.TypeFixedExpense,
				Status:      engine.StatusProjected,
				DueDate:     now,
			}
			_, _ = h.transactionRepo.Create(ctx, tx)
			monthlyEssentialCost = monthlyEssentialCost.Add(exp.Amount)
		}
	}

	// 4. Cálculo de Reserva de Emergência
	suggested6x := monthlyEssentialCost.Multiply(6)
	suggested12x := monthlyEssentialCost.Multiply(12)

	chosenMonths := req.EmergencyFundMonths
	if chosenMonths != 6 && chosenMonths != 12 {
		chosenMonths = 6
	}

	chosenTarget := suggested6x
	if chosenMonths == 12 {
		chosenTarget = suggested12x
	}

	// 5. Configura ciclo e teto flexível
	cycleDay := req.CycleStartDay
	if cycleDay < 1 || cycleDay > 28 {
		cycleDay = 1
	}

	flexCap := req.FlexibleBudgetCap
	if flexCap.IsZero() || flexCap.IsNegative() {
		flexCap = money.Max(req.InitialBalance.Subtract(req.TargetSavings).Subtract(monthlyEssentialCost), money.Zero())
	}

	_ = h.userRepo.UpdateFinancialProfile(ctx, u.ID, req.TargetSavings, flexCap, chosenTarget, chosenMonths, cycleDay)

	// 6. Cadastra investimentos iniciais
	for _, inv := range req.Investments {
		if inv.Quantity.IsPositive() && !inv.AveragePrice.IsNegative() {
			_, _ = h.investService.AddLot(ctx, u.ID, inv.Ticker, inv.Quantity, inv.AveragePrice)
		}
	}

	// 7. Roda o motor de S2S para hoje
	cycleInterval := dateinterval.CycleOf(now, cycleDay)
	snapshots, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidBalance, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	engineResult := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       now,
		TargetSavings:     req.TargetSavings,
		FlexibleBudgetCap: flexCap,
	}, engine.CurrentState{
		LiquidBalance: liquidBalance,
		Transactions:  snapshots,
	})

	// 8. Métricas de cobertura de reserva
	fundBalanceForMetrics := req.CurrentEmergencyFund
	if fundBalanceForMetrics.IsZero() {
		fundBalanceForMetrics = liquidBalance
	}

	var progressPercent decimal.Decimal
	if !chosenTarget.IsZero() {
		progressPercent = fundBalanceForMetrics.PercentOver(chosenTarget)
	}

	var monthsCovered float64
	if !monthlyEssentialCost.IsZero() {
		monthsCovered, _ = fundBalanceForMetrics.RatioOver(monthlyEssentialCost).Float64()
	}

	portfolio, _ := h.investService.List(ctx, u.ID)
	totalInvested := money.Zero()
	if portfolio != nil {
		totalInvested = portfolio.TotalInvested
	}

	resp := OnboardingResponse{
		Message: "Onboarding concluído com sucesso",
		UserID:  u.ID,
		Cycle: CycleResponse{
			StartDate:     cycleInterval.StartDate().Format("2006-01-02"),
			EndDate:       cycleInterval.EndDate().Format("2006-01-02"),
			DaysRemaining: engineResult.DaysRemaining,
			S2SToday:      engineResult.S2SToday,
			HealthStatus:  engineResult.HealthStatus,
		},
		EmergencyFund: EmergencyFundResponse{
			MonthlyEssentialCost: monthlyEssentialCost,
			Suggested6x:          suggested6x,
			Suggested12x:         suggested12x,
			ChosenTarget:         chosenTarget,
			ChosenMonths:         chosenMonths,
			CurrentBalance:       req.CurrentEmergencyFund,
			MonthsCovered:        monthsCovered,
			ProgressPercent:      progressPercent,
		},
		TotalLiquidBalance: liquidBalance,
		TotalInvested:      totalInvested,
		TotalNetWorth:      liquidBalance.Add(totalInvested),
	}

	web.JSON(w, http.StatusOK, resp)
}
