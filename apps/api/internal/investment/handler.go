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
	Ticker   string          `json:"ticker"`
	Quantity decimal.Decimal `json:"quantity"`
	Price    money.Money     `json:"price"`
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

	inv, err := h.service.AddLot(r.Context(), u.ID, req.Ticker, req.Quantity, req.Price)
	if err != nil {
		switch err {
		case ErrInvalidTicker, ErrInvalidQuantity, ErrInvalidPrice:
			web.Error(w, http.StatusBadRequest, err.Error())
		default:
			web.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	web.JSON(w, http.StatusCreated, inv)
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
