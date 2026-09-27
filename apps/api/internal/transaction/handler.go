package transaction

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/category"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(protected chi.Router) {
		protected.Use(authMiddleware)
		protected.Post("/transactions", h.handleCreate)
		protected.Get("/transactions", h.handleList)
		protected.Get("/transactions/{id}", h.handleFind)
		protected.Put("/transactions/{id}", h.handleUpdate)
		protected.Delete("/transactions/{id}", h.handleDelete)
		protected.Post("/transactions/bundles", h.handleCreateBundle)
		protected.Post("/transactions/check-in", h.handleCheckIn)
		protected.Post("/transactions/simulations", h.handleSimulate)
	})
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("data deve estar no formato AAAA-MM-DD: %s", s)
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatOptDate(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

type TransactionResponse struct {
	ID                uuid.UUID                `json:"id"`
	Description       string                   `json:"description"`
	Amount            money.Money              `json:"amount"`
	Type              engine.TransactionType   `json:"type"`
	Status            engine.TransactionStatus `json:"status"`
	CategoryID        *uuid.UUID               `json:"category_id"`
	BankAccountID     *uuid.UUID               `json:"bank_account_id"`
	DueDate           string                   `json:"due_date"`
	PaymentDate       *string                  `json:"payment_date"`
	BundleID          *uuid.UUID               `json:"bundle_id"`
	InstallmentNumber *int                     `json:"installment_number"`
	TotalInstallments *int                     `json:"total_installments"`
}

func toTransactionResponse(t Transaction) TransactionResponse {
	return TransactionResponse{
		ID:                t.ID,
		Description:       t.Description,
		Amount:            t.Amount,
		Type:              t.Type,
		Status:            t.Status,
		CategoryID:        &t.CategoryID,
		BankAccountID:     t.BankAccountID,
		DueDate:           formatDate(t.DueDate),
		PaymentDate:       formatOptDate(t.PaymentDate),
		BundleID:          t.BundleID,
		InstallmentNumber: t.InstallmentNumber,
		TotalInstallments: t.TotalInstallments,
	}
}

type createTxRequest struct {
	Description   string      `json:"description"`
	Amount        money.Money `json:"amount"`
	Type          string      `json:"type"`
	DueDate       string      `json:"due_date"`
	CategoryID    uuid.UUID   `json:"category_id"`
	BankAccountID *uuid.UUID  `json:"bank_account_id"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req createTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	dueDate, err := parseDate(req.DueDate)
	if err != nil || dueDate.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data de vencimento é obrigatória")
		return
	}

	tx, err := h.service.CreateTransaction(
		r.Context(),
		u.ID,
		req.Description,
		req.Amount,
		engine.TransactionType(req.Type),
		dueDate,
		req.CategoryID,
		req.BankAccountID,
	)
	if err != nil {
		web.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	web.JSON(w, http.StatusCreated, toTransactionResponse(*tx))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	list, err := h.service.ListTransactions(r.Context(), u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao listar transações")
		return
	}

	responses := make([]TransactionResponse, len(list))
	for i, t := range list {
		responses[i] = toTransactionResponse(t)
	}

	web.JSON(w, http.StatusOK, responses)
}

func (h *Handler) handleFind(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de transação inválido")
		return
	}

	tx, err := h.service.FindTransaction(r.Context(), id, u.ID)
	if err != nil {
		if err == ErrTransactionNotFound {
			web.Error(w, http.StatusNotFound, "Transação não encontrada")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, toTransactionResponse(*tx))
}

type updateTxRequest struct {
	Description   string      `json:"description"`
	Amount        money.Money `json:"amount"`
	Type          string      `json:"type"`
	DueDate       string      `json:"due_date"`
	CategoryID    uuid.UUID   `json:"category_id"`
	BankAccountID *uuid.UUID  `json:"bank_account_id"`
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de transação inválido")
		return
	}

	var req updateTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	dueDate, err := parseDate(req.DueDate)
	if err != nil || dueDate.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data de vencimento é obrigatória")
		return
	}

	tx, err := h.service.UpdateTransaction(
		r.Context(),
		id,
		u.ID,
		req.Description,
		req.Amount,
		engine.TransactionType(req.Type),
		dueDate,
		req.CategoryID,
		req.BankAccountID,
	)
	if err != nil {
		if err == ErrTransactionNotFound {
			web.Error(w, http.StatusNotFound, "Transação não encontrada")
			return
		}
		web.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, toTransactionResponse(*tx))
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de transação inválido")
		return
	}

	if err := h.service.DeleteTransaction(r.Context(), id, u.ID); err != nil {
		if err == ErrTransactionNotFound {
			web.Error(w, http.StatusNotFound, "Transação não encontrada")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.NoContent(w)
}

type createBundleRequest struct {
	Description       string      `json:"description"`
	TotalAmount       money.Money `json:"total_amount"`
	TotalInstallments int         `json:"total_installments"`
	FirstDueDate      string      `json:"first_due_date"`
	CategoryID        uuid.UUID   `json:"category_id"`
	BankAccountID     *uuid.UUID  `json:"bank_account_id"`
}

type BundleResponse struct {
	ID                uuid.UUID   `json:"id"`
	Description       string      `json:"description"`
	TotalAmount       money.Money `json:"total_amount"`
	TotalInstallments int         `json:"total_installments"`
	FirstDueDate      string      `json:"first_due_date"`
}

func (h *Handler) handleCreateBundle(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req createBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	firstDueDate, err := parseDate(req.FirstDueDate)
	if err != nil || firstDueDate.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data do primeiro vencimento é obrigatória")
		return
	}

	bundle, _, err := h.service.CreateBundle(
		r.Context(),
		u.ID,
		req.Description,
		req.TotalAmount,
		req.TotalInstallments,
		firstDueDate,
		req.CategoryID,
		req.BankAccountID,
	)
	if err != nil {
		web.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	web.JSON(w, http.StatusCreated, BundleResponse{
		ID:                bundle.ID,
		Description:       bundle.Description,
		TotalAmount:       bundle.TotalAmount,
		TotalInstallments: bundle.TotalInstallments,
		FirstDueDate:      formatDate(bundle.FirstDueDate),
	})
}

type checkInRequest struct {
	Date                           string       `json:"date"`
	LiquidBalance                  *money.Money `json:"liquid_balance"`
	TargetSavings                  *money.Money `json:"target_savings"`
	FlexibleBudgetCap              *money.Money `json:"flexible_budget_cap"`
	UntrackedExpenses              []struct {
		Description string      `json:"description"`
		Amount      money.Money `json:"amount"`
		CategoryID  uuid.UUID   `json:"category_id"`
	} `json:"untracked_expenses"`
	ConfirmedPendingTransactionIDs []uuid.UUID `json:"confirmed_pending_transaction_ids"`
}

func (h *Handler) handleCheckIn(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req checkInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	date, err := parseDate(req.Date)
	if err != nil || date.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data do check-in é obrigatória")
		return
	}

	liquidBalance := money.Zero()
	if req.LiquidBalance != nil {
		liquidBalance = *req.LiquidBalance
	}

	targetSavings := money.Zero()
	if req.TargetSavings != nil {
		targetSavings = *req.TargetSavings
	}

	flexibleBudgetCap := money.Zero()
	if req.FlexibleBudgetCap != nil {
		flexibleBudgetCap = *req.FlexibleBudgetCap
	}

	expenses := make([]UntrackedExpenseInput, len(req.UntrackedExpenses))
	for i, e := range req.UntrackedExpenses {
		expenses[i] = UntrackedExpenseInput{
			Description: e.Description,
			Amount:      e.Amount,
			CategoryID:  e.CategoryID,
		}
	}

	result, err := h.service.DailyCheckIn(r.Context(), u.ID, DailyCheckInInput{
		Date:                           date,
		LiquidBalance:                  liquidBalance,
		TargetSavings:                  targetSavings,
		FlexibleBudgetCap:              flexibleBudgetCap,
		UntrackedExpenses:              expenses,
		ConfirmedPendingTransactionIDs: req.ConfirmedPendingTransactionIDs,
	})
	if err != nil {
		if err == category.ErrCategoryNotFound {
			web.Error(w, http.StatusBadRequest, "Categoria não encontrada")
			return
		}
		web.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, result)
}

type simulationRequest struct {
	Date              string       `json:"date"`
	LiquidBalance     *money.Money `json:"liquid_balance"`
	TargetSavings     *money.Money `json:"target_savings"`
	FlexibleBudgetCap *money.Money `json:"flexible_budget_cap"`
	TotalAmount       money.Money  `json:"total_amount"`
	Installments      int          `json:"installments"`
	FirstDueDate      string       `json:"first_due_date"`
}

func (h *Handler) handleSimulate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req simulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	date, err := parseDate(req.Date)
	if err != nil || date.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data da simulação é obrigatória")
		return
	}

	firstDueDate, err := parseDate(req.FirstDueDate)
	if err != nil || firstDueDate.IsZero() {
		web.Error(w, http.StatusBadRequest, "Data do primeiro vencimento é obrigatória")
		return
	}

	liquidBalance := money.Zero()
	if req.LiquidBalance != nil {
		liquidBalance = *req.LiquidBalance
	}

	targetSavings := money.Zero()
	if req.TargetSavings != nil {
		targetSavings = *req.TargetSavings
	}

	flexibleBudgetCap := money.Zero()
	if req.FlexibleBudgetCap != nil {
		flexibleBudgetCap = *req.FlexibleBudgetCap
	}

	cycles, err := h.service.SimulatePurchase(r.Context(), u.ID, SimulationInput{
		Date:              date,
		LiquidBalance:     liquidBalance,
		TargetSavings:     targetSavings,
		FlexibleBudgetCap: flexibleBudgetCap,
		TotalAmount:       req.TotalAmount,
		Installments:      req.Installments,
		FirstDueDate:      firstDueDate,
	})
	if err != nil {
		web.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, map[string]any{"cycles": cycles})
}
