package botapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
)

type TransactionSummaryItem struct {
	ID           uuid.UUID              `json:"id"`
	Description  string                 `json:"description"`
	Amount       money.Money            `json:"amount"`
	CategoryName string                 `json:"category_name"`
	Type         engine.TransactionType `json:"type"`
	Date         string                 `json:"date"`
}

type DaySummaryResponse struct {
	Date              string                   `json:"date"`
	TotalSpent        money.Money              `json:"total_spent"`
	TransactionsCount int                      `json:"transactions_count"`
	Transactions      []TransactionSummaryItem `json:"transactions"`
}

type CategorySummaryItem struct {
	CategoryID        uuid.UUID       `json:"category_id"`
	CategoryName      string          `json:"category_name"`
	Amount            money.Money     `json:"amount"`
	Percentage        decimal.Decimal `json:"percentage"`
	TransactionsCount int             `json:"transactions_count"`
}

type WeekSummaryResponse struct {
	StartDate         string                `json:"start_date"`
	EndDate           string                `json:"end_date"`
	DaysElapsed       int                   `json:"days_elapsed"`
	TotalSpent        money.Money           `json:"total_spent"`
	DailyAverage      money.Money           `json:"daily_average"`
	TransactionsCount int                   `json:"transactions_count"`
	TopCategories     []CategorySummaryItem `json:"top_categories"`
}

type SummaryReportResponse struct {
	Period             string               `json:"period"`
	AccountBalance     money.Money          `json:"account_balance"`
	TotalLiquidBalance money.Money          `json:"total_liquid_balance"`
	S2SToday           money.Money          `json:"s2s_today"`
	HealthStatus       engine.HealthStatus  `json:"health_status"`
	DaysRemaining      int                  `json:"days_remaining"`
	Today              *DaySummaryResponse  `json:"today,omitempty"`
	Week               *WeekSummaryResponse `json:"week,omitempty"`
}

func (h *Handler) handleSummaryReport(w http.ResponseWriter, r *http.Request) {
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

	period := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("period")))
	if period == "" {
		period = "today"
	}
	if period != "today" && period != "week" {
		web.Error(w, http.StatusBadRequest, "período inválido. Valores aceitos: 'today' ou 'week'")
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

	refDate := time.Now().UTC()
	dateParam := r.URL.Query().Get("date")
	if dateParam != "" {
		if parsed, err := time.Parse("2006-01-02", dateParam); err == nil {
			refDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
	}

	accounts, _ := h.bankAccountRepo.ListByUserID(ctx, u.ID)
	totalLiquid, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	accountBalance := totalLiquid
	if len(accounts) > 0 {
		accountBalance = accounts[0].CurrentBalance
	}

	cycleInterval := dateinterval.CycleOf(refDate, u.CycleStartDay)
	snapshots, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)

	flexCap := u.FlexibleBudgetCap
	if flexCap.IsZero() {
		flexCap = totalLiquid
	}

	engineResult := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       refDate,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCap,
	}, engine.CurrentState{
		LiquidBalance: totalLiquid,
		Transactions:  snapshots,
	})

	resp := SummaryReportResponse{
		Period:             period,
		AccountBalance:     accountBalance,
		TotalLiquidBalance: totalLiquid,
		S2SToday:           engineResult.S2SToday,
		HealthStatus:       engineResult.HealthStatus,
		DaysRemaining:      engineResult.DaysRemaining,
	}

	if period == "today" {
		dailySum, err := h.transactionRepo.GetDailyExpensesSummary(ctx, u.ID, refDate)
		if err != nil {
			web.Error(w, http.StatusInternalServerError, "Erro ao consultar extrato diário")
			return
		}

		txItems := make([]TransactionSummaryItem, len(dailySum.Transactions))
		for i, t := range dailySum.Transactions {
			txItems[i] = TransactionSummaryItem{
				ID:           t.ID,
				Description:  t.Description,
				Amount:       t.Amount,
				CategoryName: t.CategoryName,
				Type:         t.Type,
				Date:         t.Date.Format("2006-01-02"),
			}
		}

		resp.Today = &DaySummaryResponse{
			Date:              dailySum.Date.Format("2006-01-02"),
			TotalSpent:        dailySum.TotalSpent,
			TransactionsCount: dailySum.Count,
			Transactions:      txItems,
		}
	} else if period == "week" {
		weekInterval, daysElapsed := dateinterval.WeekOf(refDate)
		weeklySum, err := h.transactionRepo.GetWeeklyExpensesSummary(ctx, u.ID, weekInterval.StartDate(), weekInterval.EndDate(), daysElapsed)
		if err != nil {
			web.Error(w, http.StatusInternalServerError, "Erro ao consultar resumo semanal")
			return
		}

		catItems := make([]CategorySummaryItem, len(weeklySum.TopCategories))
		for i, c := range weeklySum.TopCategories {
			catItems[i] = CategorySummaryItem{
				CategoryID:        c.CategoryID,
				CategoryName:      c.CategoryName,
				Amount:            c.Amount,
				Percentage:        c.Percentage,
				TransactionsCount: c.Count,
			}
		}

		resp.Week = &WeekSummaryResponse{
			StartDate:         weeklySum.StartDate.Format("2006-01-02"),
			EndDate:           weeklySum.EndDate.Format("2006-01-02"),
			DaysElapsed:       weeklySum.DaysElapsed,
			TotalSpent:        weeklySum.TotalSpent,
			DailyAverage:      weeklySum.DailyAverage,
			TransactionsCount: weeklySum.Count,
			TopCategories:     catItems,
		}
	}

	web.JSON(w, http.StatusOK, resp)
}
