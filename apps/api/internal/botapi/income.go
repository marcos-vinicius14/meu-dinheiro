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

type IncomeRequest struct {
	TelegramID   int64       `json:"telegram_id"`
	Amount       money.Money `json:"amount"`
	Description  string      `json:"description"`
	CategoryName string      `json:"category_name,omitempty"`
	Date         string      `json:"date,omitempty"`
}

type IncomeResponse struct {
	Transaction        *transaction.Transaction `json:"transaction"`
	CategoryName       string                   `json:"category_name"`
	PreviousS2S        money.Money              `json:"previous_s2s"`
	NewS2S             money.Money              `json:"new_s2s"`
	HealthStatus       engine.HealthStatus      `json:"health_status"`
	DaysRemaining      int                      `json:"days_remaining"`
	TotalLiquidBalance money.Money              `json:"total_liquid_balance"`
}

func (h *Handler) handleIncome(w http.ResponseWriter, r *http.Request) {
	var req IncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}
	if !req.Amount.IsPositive() {
		web.Error(w, http.StatusBadRequest, "Valor da receita deve ser maior que zero")
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

	txDate := time.Now().UTC()
	if req.Date != "" {
		if parsed, err := time.Parse("2006-01-02", req.Date); err == nil {
			txDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
	}

	categories, err := h.categoryRepo.ListByUserID(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao carregar categorias")
		return
	}

	catName := strings.TrimSpace(req.CategoryName)
	if catName == "" {
		catName = "Renda"
	}

	catID := matchCategory(categories, catName)
	if catID == uuid.Nil && len(categories) > 0 {
		catID = categories[0].ID
		catName = categories[0].Description
	}

	var bankAccountID *uuid.UUID
	accounts, err := h.bankAccountRepo.ListByUserID(ctx, u.ID)
	if err == nil && len(accounts) > 0 {
		bankAccountID = &accounts[0].ID
	}

	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		desc = "Entrada de dinheiro"
	}

	// 1. Calcula S2S antes da entrada
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

	// 2. Insere a transação de INCOME com status CONFIRMED
	tx := &transaction.Transaction{
		UserID:        u.ID,
		CategoryID:    catID,
		BankAccountID: bankAccountID,
		Description:   desc,
		Amount:        req.Amount,
		Type:          engine.TypeIncome,
		Status:        engine.StatusConfirmed,
		DueDate:       txDate,
		PaymentDate:   &txDate,
	}

	createdTx, err := h.transactionRepo.Create(ctx, tx)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao criar transação de receita")
		return
	}

	// 3. Calcula novo S2S após a receita
	snapshotsAfter, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidAfter, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	flexCapAfter := u.FlexibleBudgetCap
	if flexCapAfter.IsZero() {
		flexCapAfter = liquidAfter
	}

	engAfter := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       txDate,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCapAfter,
	}, engine.CurrentState{
		LiquidBalance: liquidAfter,
		Transactions:  snapshotsAfter,
	})

	web.JSON(w, http.StatusCreated, IncomeResponse{
		Transaction:        createdTx,
		CategoryName:       catName,
		PreviousS2S:        engBefore.S2SToday,
		NewS2S:             engAfter.S2SToday,
		HealthStatus:       engAfter.HealthStatus,
		DaysRemaining:      engAfter.DaysRemaining,
		TotalLiquidBalance: liquidAfter,
	})
}
