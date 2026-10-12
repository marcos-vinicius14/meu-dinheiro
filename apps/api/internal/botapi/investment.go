package botapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

type AddInvestmentRequest struct {
	TelegramID   int64           `json:"telegram_id"`
	Ticker       string          `json:"ticker"`
	Quantity     decimal.Decimal `json:"quantity"`
	Price        money.Money     `json:"price"`
	AccountID    *uuid.UUID      `json:"account_id,omitempty"`
	DebitAccount *bool           `json:"debit_account,omitempty"`
}

type AddInvestmentResponse struct {
	Investment         *investment.Investment `json:"investment"`
	DebitedAmount      money.Money            `json:"debited_amount"`
	AccountID          uuid.UUID              `json:"account_id,omitempty"`
	AccountName        string                 `json:"account_name,omitempty"`
	AccountBalance     money.Money            `json:"account_balance"`
	TotalLiquidBalance money.Money            `json:"total_liquid_balance"`
	TotalInvested      money.Money            `json:"total_invested"`
	TotalNetWorth      money.Money            `json:"total_net_worth"`
	PreviousS2S        money.Money            `json:"previous_s2s"`
	NewS2S             money.Money            `json:"new_s2s"`
	DaysRemaining      int                    `json:"days_remaining"`
}

type ListInvestmentsResponse struct {
	Investments        []investment.Investment `json:"investments"`
	TotalInvested      money.Money             `json:"total_invested"`
	TotalLiquidBalance money.Money             `json:"total_liquid_balance"`
	TotalNetWorth      money.Money             `json:"total_net_worth"`
}

type SellInvestmentRequest struct {
	TelegramID int64           `json:"telegram_id"`
	Ticker     string          `json:"ticker"`
	Quantity   decimal.Decimal `json:"quantity"`
}

type SellInvestmentResponse struct {
	Investment         *investment.Investment `json:"investment"`
	Ticker             string                 `json:"ticker"`
	SoldQuantity       decimal.Decimal        `json:"sold_quantity"`
	RemainingQuantity  decimal.Decimal        `json:"remaining_quantity"`
	AveragePrice       money.Money            `json:"average_price"`
	IsClosed           bool                   `json:"is_closed"`
	TotalInvested      money.Money            `json:"total_invested"`
	TotalLiquidBalance money.Money            `json:"total_liquid_balance"`
	TotalNetWorth      money.Money            `json:"total_net_worth"`
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
		if errors.Is(err, user.ErrUserNotFound) {
			web.Error(w, http.StatusNotFound, "Usuário não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		return
	}

	// 1. Calcula S2S antes da operação
	now := time.Now().UTC()
	cycleInterval := dateinterval.CycleOf(now, u.CycleStartDay)
	snapshotsBefore, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidBefore, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	flexCapBefore := u.FlexibleBudgetCap
	if flexCapBefore.IsZero() {
		flexCapBefore = liquidBefore
	}

	engBefore := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       now,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCapBefore,
	}, engine.CurrentState{
		LiquidBalance: liquidBefore,
		Transactions:  snapshotsBefore,
	})

	var inv *investment.Investment
	var debitedAmount money.Money
	var accountID uuid.UUID
	var accountName string
	var accountBalance money.Money

	debitAccount := true
	if req.DebitAccount != nil {
		debitAccount = *req.DebitAccount
	}

	if debitAccount {
		result, err := h.investService.AddLotWithDebit(ctx, u.ID, req.AccountID, req.Ticker, req.Quantity, req.Price)
		if err != nil {
			switch {
			case errors.Is(err, investment.ErrInvalidTicker),
				errors.Is(err, investment.ErrInvalidQuantity),
				errors.Is(err, investment.ErrInvalidPrice),
				errors.Is(err, investment.ErrInsufficientBalance):
				web.Error(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, investment.ErrNoAccountAvailable),
				errors.Is(err, bankaccount.ErrAccountNotFound):
				web.Error(w, http.StatusNotFound, err.Error())
			default:
				web.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		inv = result.Investment
		debitedAmount = result.DebitedAmount
		accountID = result.AccountID
		accountName = result.AccountName
		accountBalance = result.AccountBalance
	} else {
		invResult, err := h.investService.AddLot(ctx, u.ID, req.Ticker, req.Quantity, req.Price)
		if err != nil {
			switch err {
			case investment.ErrInvalidTicker, investment.ErrInvalidQuantity, investment.ErrInvalidPrice:
				web.Error(w, http.StatusBadRequest, err.Error())
			default:
				web.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		inv = invResult
	}

	// 2. Calcula S2S e liquidez após a operação
	snapshotsAfter, _ := h.transactionRepo.LoadSnapshotsByUserID(ctx, u.ID)
	liquidAfter, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)

	flexCapAfter := u.FlexibleBudgetCap
	if flexCapAfter.IsZero() {
		flexCapAfter = liquidAfter
	}

	engAfter := h.predictiveEng.Calculate(engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       now,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCapAfter,
	}, engine.CurrentState{
		LiquidBalance: liquidAfter,
		Transactions:  snapshotsAfter,
	})

	portfolio, _ := h.investService.List(ctx, u.ID)

	web.JSON(w, http.StatusOK, AddInvestmentResponse{
		Investment:         inv,
		DebitedAmount:      debitedAmount,
		AccountID:          accountID,
		AccountName:        accountName,
		AccountBalance:     accountBalance,
		TotalLiquidBalance: liquidAfter,
		TotalInvested:      portfolio.TotalInvested,
		TotalNetWorth:      liquidAfter.Add(portfolio.TotalInvested),
		PreviousS2S:        engBefore.S2SToday,
		NewS2S:             engAfter.S2SToday,
		DaysRemaining:      engAfter.DaysRemaining,
	})
}

func (h *Handler) handleListInvestments(w http.ResponseWriter, r *http.Request) {
	telegramIDStr := r.URL.Query().Get("telegram_id")
	tgID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil || tgID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}

	ctx := r.Context()
	u, err := h.userRepo.FindByTelegramID(ctx, tgID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			web.Error(w, http.StatusNotFound, "Usuário não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		return
	}

	portfolio, err := h.investService.List(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalLiquid, err := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalNetWorth := totalLiquid.Add(portfolio.TotalInvested)

	web.JSON(w, http.StatusOK, ListInvestmentsResponse{
		Investments:        portfolio.Investments,
		TotalInvested:      portfolio.TotalInvested,
		TotalLiquidBalance: totalLiquid,
		TotalNetWorth:      totalNetWorth,
	})
}

func (h *Handler) handleSellInvestment(w http.ResponseWriter, r *http.Request) {
	var req SellInvestmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 || strings.TrimSpace(req.Ticker) == "" {
		web.Error(w, http.StatusBadRequest, "telegram_id e ticker são obrigatórios")
		return
	}
	if !req.Quantity.IsPositive() {
		web.Error(w, http.StatusBadRequest, "quantidade deve ser maior que zero")
		return
	}

	ctx := r.Context()
	u, err := h.userRepo.FindByTelegramID(ctx, req.TelegramID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			web.Error(w, http.StatusNotFound, "Usuário não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar usuário")
		return
	}

	inv, err := h.investService.Sell(ctx, u.ID, req.Ticker, req.Quantity)
	if err != nil {
		if errors.Is(err, investment.ErrInvestmentNotFound) {
			web.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, investment.ErrInsufficientQuantity) ||
			errors.Is(err, investment.ErrInvalidTicker) ||
			errors.Is(err, investment.ErrInvalidQuantity) {
			web.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	portfolio, _ := h.investService.List(ctx, u.ID)
	totalLiquid, _ := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)
	totalNetWorth := totalLiquid.Add(portfolio.TotalInvested)
	isClosed := inv.Quantity.IsZero()

	web.JSON(w, http.StatusOK, SellInvestmentResponse{
		Investment:         inv,
		Ticker:             inv.Ticker,
		SoldQuantity:       req.Quantity,
		RemainingQuantity:  inv.Quantity,
		AveragePrice:       inv.AveragePrice,
		IsClosed:           isClosed,
		TotalInvested:      portfolio.TotalInvested,
		TotalLiquidBalance: totalLiquid,
		TotalNetWorth:      totalNetWorth,
	})
}
