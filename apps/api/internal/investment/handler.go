package investment

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
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
		protected.Get("/investments", h.handleList)
		protected.Post("/investments", h.handleAddLot)
		protected.Delete("/investments/{id}", h.handleDelete)
	})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	portfolio, err := h.service.List(r.Context(), u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao listar investimentos")
		return
	}

	web.JSON(w, http.StatusOK, portfolio)
}

type addLotRequest struct {
	Ticker    string          `json:"ticker"`
	Quantity  decimal.Decimal `json:"quantity"`
	Price     money.Money     `json:"price"`
	AccountID *uuid.UUID      `json:"account_id,omitempty"`
}

func (h *Handler) handleAddLot(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req addLotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	result, err := h.service.AddLotWithDebit(r.Context(), u.ID, req.AccountID, req.Ticker, req.Quantity, req.Price)
	if err != nil {
		switch err {
		case ErrInvalidTicker, ErrInvalidQuantity, ErrInvalidPrice, ErrInsufficientBalance:
			web.Error(w, http.StatusBadRequest, err.Error())
		case ErrNoAccountAvailable:
			web.Error(w, http.StatusNotFound, err.Error())
		default:
			web.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	web.JSON(w, http.StatusCreated, result.Investment)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de investimento inválido")
		return
	}

	if err := h.service.Delete(r.Context(), id, u.ID); err != nil {
		if err == ErrInvestmentNotFound {
			web.Error(w, http.StatusNotFound, "Investimento não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.NoContent(w)
}
