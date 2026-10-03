package botapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/investment"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
)

type AddInvestmentRequest struct {
	TelegramID int64           `json:"telegram_id"`
	Ticker     string          `json:"ticker"`
	Quantity   decimal.Decimal `json:"quantity"`
	Price      money.Money     `json:"price"`
}

type AddInvestmentResponse struct {
	Investment    *investment.Investment `json:"investment"`
	TotalInvested money.Money            `json:"total_invested"`
	TotalNetWorth money.Money            `json:"total_net_worth"`
}

func (h *Handler) handleAddInvestment(w http.ResponseWriter, r *http.Request) {
	var req AddInvestmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 || strings.TrimSpace(req.Ticker) == "" {
		web.Error(w, http.StatusBadRequest, "telegram_id e ticker são obrigatórios")
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

	inv, err := h.investService.AddLot(ctx, u.ID, req.Ticker, req.Quantity, req.Price)
	if err != nil {
		switch err {
		case investment.ErrInvalidTicker, investment.ErrInvalidQuantity, investment.ErrInvalidPrice:
			web.Error(w, http.StatusBadRequest, err.Error())
		default:
			web.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	portfolio, _ := h.investService.List(ctx, u.ID)
	totalLiquid, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	web.JSON(w, http.StatusOK, AddInvestmentResponse{
		Investment:    inv,
		TotalInvested: portfolio.TotalInvested,
		TotalNetWorth: totalLiquid.Add(portfolio.TotalInvested),
	})
}
