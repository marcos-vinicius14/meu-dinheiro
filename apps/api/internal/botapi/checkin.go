package botapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type CheckInExpenseInput struct {
	Description  string      `json:"description"`
	Amount       money.Money `json:"amount"`
	CategoryID   *uuid.UUID  `json:"category_id,omitempty"`
	CategoryName string      `json:"category_name,omitempty"`
}

type CheckInRequest struct {
	TelegramID        int64                 `json:"telegram_id"`
	Date              string                `json:"date,omitempty"`
	UntrackedExpenses []CheckInExpenseInput `json:"untracked_expenses,omitempty"`
}

type CheckInResponse struct {
	S2SCalculated        money.Money         `json:"s2s_calculated"`
	SpentToday           money.Money         `json:"spent_today"`
	DailyQuota           money.Money         `json:"daily_quota"`
	DeltaSavings         money.Money         `json:"delta_savings"`
	HealthStatus         engine.HealthStatus `json:"health_status"`
	NextDayS2S           money.Money         `json:"next_day_s2s"`
	ProjectedFreeBalance money.Money         `json:"projected_free_balance"`
	DaysRemaining        int                 `json:"days_remaining"`
	Message              string              `json:"message"`
}

func (h *Handler) handleDailyCheckIn(w http.ResponseWriter, r *http.Request) {
	var req CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	if req.TelegramID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}

	ctx := r.Context()
	u, err := h.userRepo.FindByTelegramID(ctx, req.TelegramID)
	if err != nil {
		if err == user.ErrUserNotFound {
			web.Error(w, http.StatusNotFound, "Usuário não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		return
	}

	checkinDate := time.Now().UTC()
	if req.Date != "" {
		if parsed, err := time.Parse("2006-01-02", req.Date); err == nil {
			checkinDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
	}

	categories, err := h.categoryRepo.ListByUserID(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao carregar categorias")
		return
	}

	untracked := make([]transaction.UntrackedExpenseInput, len(req.UntrackedExpenses))
	for i, exp := range req.UntrackedExpenses {
		if !exp.Amount.IsPositive() {
			web.Error(w, http.StatusBadRequest, "Valor da despesa deve ser maior que zero")
			return
		}

		var catID uuid.UUID
		if exp.CategoryID != nil && *exp.CategoryID != uuid.Nil {
			catID = *exp.CategoryID
		} else {
			catID = matchCategory(categories, exp.CategoryName)
		}

		desc := strings.TrimSpace(exp.Description)
		if desc == "" {
			desc = "Despesa não registrada"
		}

		untracked[i] = transaction.UntrackedExpenseInput{
			Description: desc,
			Amount:      exp.Amount,
			CategoryID:  catID,
		}
	}

	totalLiquid, err := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao calcular saldo de liquidez")
		return
	}

	flexCap := u.FlexibleBudgetCap
	if flexCap.IsZero() {
		flexCap = totalLiquid
	}

	input := transaction.DailyCheckInInput{
		Date:                           checkinDate,
		LiquidBalance:                  totalLiquid,
		TargetSavings:                  u.TargetSavings,
		FlexibleBudgetCap:              flexCap,
		UntrackedExpenses:              untracked,
		ConfirmedPendingTransactionIDs: nil,
	}

	result, err := h.txService.DailyCheckIn(ctx, u.ID, input)
	if err != nil {
		web.Error(w, http.StatusBadRequest, fmt.Sprintf("Erro ao realizar check-in: %v", err))
		return
	}

	var message string
	if result.DeltaFromSafeToSpend.IsPositive() {
		message = fmt.Sprintf("Parabéns! Você economizou %s hoje, aumentando sua margem para amanhã.", result.DeltaFromSafeToSpend.String())
	} else if result.DeltaFromSafeToSpend.IsNegative() {
		absDelta := money.New(result.DeltaFromSafeToSpend.Decimal().Abs())
		message = fmt.Sprintf("Você ultrapassou a cota do dia em %s. O S2S foi recalculado defensivamente.", absDelta.String())
	} else {
		message = "Check-in realizado com sucesso! Seus gastos ficaram exatamente na meta do dia."
	}

	resp := CheckInResponse{
		S2SCalculated:        result.S2SCalculated,
		SpentToday:           result.SpentToday,
		DailyQuota:           result.S2SCalculated,
		DeltaSavings:         result.DeltaFromSafeToSpend,
		HealthStatus:         result.HealthStatus,
		NextDayS2S:           result.NextDayS2S,
		ProjectedFreeBalance: result.ProjectedFreeBalance,
		DaysRemaining:        result.DaysRemaining,
		Message:              message,
	}

	web.JSON(w, http.StatusOK, resp)
}
