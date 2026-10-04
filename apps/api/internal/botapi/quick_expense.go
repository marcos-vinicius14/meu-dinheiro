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
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type QuickExpenseRequest struct {
	TelegramID   int64       `json:"telegram_id"`
	Amount       money.Money `json:"amount"`
	Description  string      `json:"description"`
	CategoryID   *uuid.UUID  `json:"category_id,omitempty"`
	CategoryName string      `json:"category_name,omitempty"`
	Date         string      `json:"date,omitempty"`
}

type QuickExpenseResponse struct {
	Transaction   *transaction.Transaction `json:"transaction"`
	CategoryName  string                   `json:"category_name"`
	PreviousS2S   money.Money              `json:"previous_s2s"`
	NewS2S        money.Money              `json:"new_s2s"`
	HealthStatus  engine.HealthStatus      `json:"health_status"`
	DaysRemaining int                      `json:"days_remaining"`
}

func (h *Handler) handleQuickExpense(w http.ResponseWriter, r *http.Request) {
	var req QuickExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}
	if !req.Amount.IsPositive() {
		web.Error(w, http.StatusBadRequest, "Valor do gasto deve ser maior que zero")
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

	// Resolve data
	txDate := time.Now().UTC()
	if req.Date != "" {
		if parsed, err := time.Parse("2006-01-02", req.Date); err == nil {
			txDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
	}

	// Resolve categoria
	categories, err := h.categoryRepo.ListByUserID(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao carregar categorias")
		return
	}

	var catID uuid.UUID
	var catName string

	if req.CategoryID != nil {
		for _, c := range categories {
			if c.ID == *req.CategoryID {
				catID = c.ID
				catName = c.Description
				break
			}
		}
	}

	if catID == uuid.Nil {
		catID = matchCategory(categories, req.CategoryName)
		for _, c := range categories {
			if c.ID == catID {
				catName = c.Description
				break
			}
		}
	}

	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		desc = "Despesa rápida"
	}

	// 1. Calcula S2S antes da despesa
	cycleInterval := dateinterval.CycleOf(txDate, u.CycleStartDay)
	snapshotsBefore, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidBefore, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	flexCap := u.FlexibleBudgetCap
	if flexCap.IsZero() {
		flexCap = liquidBefore
	}

	engBefore := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       txDate,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCap,
	}, engine.CurrentState{
		LiquidBalance: liquidBefore,
		Transactions:  snapshotsBefore,
	})

	// 2. Insere a despesa rápida como FLEXIBLE_EXPENSE CONFIRMED
	tx := &transaction.Transaction{
		UserID:      u.ID,
		CategoryID:  catID,
		Description: desc,
		Amount:      req.Amount,
		Type:        engine.TypeFlexibleExpense,
		Status:      engine.StatusConfirmed,
		DueDate:     txDate,
		PaymentDate: &txDate,
	}

	createdTx, err := h.transactionRepo.Create(ctx, tx)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao criar transação")
		return
	}

	// 3. Calcula novo S2S após a despesa
	snapshotsAfter, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidAfter, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	engAfter := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       txDate,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCap,
	}, engine.CurrentState{
		LiquidBalance: liquidAfter,
		Transactions:  snapshotsAfter,
	})

	web.JSON(w, http.StatusCreated, QuickExpenseResponse{
		Transaction:   createdTx,
		CategoryName:  catName,
		PreviousS2S:   engBefore.S2SToday,
		NewS2S:        engAfter.S2SToday,
		HealthStatus:  engAfter.HealthStatus,
		DaysRemaining: engAfter.DaysRemaining,
	})
}
