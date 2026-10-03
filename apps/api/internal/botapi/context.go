package botapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/investment"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
)

type UserFinancialContext struct {
	User struct {
		ID                  uuid.UUID   `json:"id"`
		TelegramID          int64       `json:"telegram_id"`
		FirstName           string      `json:"first_name"`
		Username            *string     `json:"username,omitempty"`
		TargetSavings       money.Money `json:"target_savings"`
		FlexibleBudgetCap   money.Money `json:"flexible_budget_cap"`
		EmergencyFundTarget money.Money `json:"emergency_fund_target"`
		EmergencyFundMonths int         `json:"emergency_fund_months"`
		CycleStartDay       int         `json:"cycle_start_day"`
	} `json:"user"`
	Accounts           []bankaccount.BankAccount `json:"accounts"`
	TotalLiquidBalance money.Money               `json:"total_liquid_balance"`
	Cycle              struct {
		StartDate         string              `json:"start_date"`
		EndDate           string              `json:"end_date"`
		DaysRemaining     int                 `json:"days_remaining"`
		S2SToday          money.Money         `json:"s2s_today"`
		HealthStatus      engine.HealthStatus `json:"health_status"`
		ProjectedBalance  money.Money         `json:"projected_balance"`
		RemainingFlexible money.Money         `json:"remaining_flexible"`
		FlexibleSpent     money.Money         `json:"flexible_spent"`
	} `json:"cycle"`
	EmergencyFund struct {
		MonthlyEssentialCost money.Money     `json:"monthly_essential_cost"`
		Target               money.Money     `json:"target"`
		Months               int             `json:"months"`
		MonthsCovered        float64         `json:"months_covered"`
		ProgressPercent      decimal.Decimal `json:"progress_percent"`
	} `json:"emergency_fund"`
	Investments   []investment.Investment `json:"investments"`
	TotalInvested money.Money             `json:"total_invested"`
	TotalNetWorth money.Money             `json:"total_net_worth"`
}

func (h *Handler) handleContextByTelegram(w http.ResponseWriter, r *http.Request) {
	tgIDStr := r.URL.Query().Get("telegram_id")
	if tgIDStr == "" {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}

	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		web.Error(w, http.StatusBadRequest, "telegram_id inválido")
		return
	}

	ctx := r.Context()
	u, err := h.userRepo.FindByTelegramID(ctx, tgID)
	if err != nil {
		if err == user.ErrUserNotFound {
			web.Error(w, http.StatusNotFound, "Usuário não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		return
	}

	accounts, _ := h.bankAccountRepo.ListByUserID(ctx, u.ID)
	totalLiquid, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	now := time.Now().UTC()
	cycleInterval := dateinterval.CycleOf(now, u.CycleStartDay)
	snapshots, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)

	flexCap := u.FlexibleBudgetCap
	if flexCap.IsZero() {
		flexCap = totalLiquid
	}

	engineResult := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       now,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCap,
	}, engine.CurrentState{
		LiquidBalance: totalLiquid,
		Transactions:  snapshots,
	})

	essentialCost, _ := h.userRepo.CalculateMonthlyEssentialCost(ctx, u.ID)

	var progressPercent decimal.Decimal
	if !u.EmergencyFundTarget.IsZero() {
		progressPercent = totalLiquid.PercentOver(u.EmergencyFundTarget)
	}

	var monthsCovered float64
	if !essentialCost.IsZero() {
		monthsCovered, _ = totalLiquid.RatioOver(essentialCost).Float64()
	}

	portfolio, _ := h.investService.List(ctx, u.ID)
	investmentsList := []investment.Investment{}
	totalInvested := money.Zero()
	if portfolio != nil {
		investmentsList = portfolio.Investments
		totalInvested = portfolio.TotalInvested
	}

	resp := UserFinancialContext{}
	resp.User.ID = u.ID
	resp.User.TelegramID = u.TelegramID
	resp.User.FirstName = u.FirstName
	resp.User.Username = u.Username
	resp.User.TargetSavings = u.TargetSavings
	resp.User.FlexibleBudgetCap = u.FlexibleBudgetCap
	resp.User.EmergencyFundTarget = u.EmergencyFundTarget
	resp.User.EmergencyFundMonths = u.EmergencyFundMonths
	resp.User.CycleStartDay = u.CycleStartDay

	resp.Accounts = accounts
	resp.TotalLiquidBalance = totalLiquid

	resp.Cycle.StartDate = cycleInterval.StartDate().Format("2006-01-02")
	resp.Cycle.EndDate = cycleInterval.EndDate().Format("2006-01-02")
	resp.Cycle.DaysRemaining = engineResult.DaysRemaining
	resp.Cycle.S2SToday = engineResult.S2SToday
	resp.Cycle.HealthStatus = engineResult.HealthStatus
	resp.Cycle.ProjectedBalance = engineResult.ProjectedBalance
	resp.Cycle.RemainingFlexible = engineResult.RemainingFlexible
	resp.Cycle.FlexibleSpent = engineResult.FlexSpent

	resp.EmergencyFund.MonthlyEssentialCost = essentialCost
	resp.EmergencyFund.Target = u.EmergencyFundTarget
	resp.EmergencyFund.Months = u.EmergencyFundMonths
	resp.EmergencyFund.MonthsCovered = monthsCovered
	resp.EmergencyFund.ProgressPercent = progressPercent

	resp.Investments = investmentsList
	resp.TotalInvested = totalInvested
	resp.TotalNetWorth = totalLiquid.Add(totalInvested)

	web.JSON(w, http.StatusOK, resp)
}
