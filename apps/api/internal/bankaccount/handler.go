package bankaccount

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(protected chi.Router) {
		protected.Use(authMiddleware)
		protected.Post("/bank-accounts", h.handleCreate)
		protected.Get("/bank-accounts", h.handleList)
		protected.Get("/bank-accounts/{id}", h.handleFind)
		protected.Put("/bank-accounts/{id}", h.handleUpdate)
		protected.Delete("/bank-accounts/{id}", h.handleDelete)
	})
}

type createRequest struct {
	Name           string       `json:"name"`
	Type           string       `json:"type"`
	InitialBalance *money.Money `json:"initial_balance"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	initialBalance := money.Zero()
	if req.InitialBalance != nil {
		initialBalance = *req.InitialBalance
	}

	account, err := h.repo.Create(r.Context(), u.ID, req.Name, req.Type, initialBalance)
	if err != nil {
		switch err {
		case ErrAccountLimitReached:
			web.Error(w, http.StatusBadRequest, err.Error())
		case ErrInvalidAccountName, ErrInvalidAccountType:
			web.Error(w, http.StatusBadRequest, err.Error())
		default:
			web.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	web.JSON(w, http.StatusCreated, account)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	accounts, err := h.repo.ListByUserID(r.Context(), u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao listar contas bancárias")
		return
	}

	web.JSON(w, http.StatusOK, accounts)
}

func (h *Handler) handleFind(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de conta inválido")
		return
	}

	account, err := h.repo.FindByIDAndUserID(r.Context(), id, u.ID)
	if err != nil {
		if err == ErrAccountNotFound {
			web.Error(w, http.StatusNotFound, "Conta bancária não encontrada")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, account)
}

type updateRequest struct {
	Name           string      `json:"name"`
	InitialBalance money.Money `json:"initial_balance"`
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de conta inválido")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	account, err := h.repo.Update(r.Context(), id, u.ID, req.Name, req.InitialBalance)
	if err != nil {
		if err == ErrAccountNotFound {
			web.Error(w, http.StatusNotFound, "Conta bancária não encontrada")
			return
		}
		if err == ErrInvalidAccountName {
			web.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, account)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de conta inválido")
		return
	}

	err = h.repo.Delete(r.Context(), id, u.ID)
	if err != nil {
		if err == ErrAccountNotFound {
			web.Error(w, http.StatusNotFound, "Conta bancária não encontrada")
			return
		}
		if err == ErrAccountHasTransactions {
			web.Error(w, http.StatusConflict, "Conta bancária possui transações vinculadas")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.NoContent(w)
}
