package botapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type AdjustBalanceRequest struct {
	TelegramID int64       `json:"telegram_id"`
	AccountID  *uuid.UUID  `json:"account_id,omitempty"`
	NewBalance money.Money `json:"new_balance"`
}

type AdjustBalanceResponse struct {
	AccountID          uuid.UUID   `json:"account_id"`
	AccountName        string      `json:"account_name"`
	PreviousBalance    money.Money `json:"previous_balance"`
	NewBalance         money.Money `json:"new_balance"`
	TotalLiquidBalance money.Money `json:"total_liquid_balance"`
	Message            string      `json:"message"`
}

func (h *Handler) handleAdjustBalance(w http.ResponseWriter, r *http.Request) {
	var req AdjustBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	if req.TelegramID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}

	if req.NewBalance.IsNegative() {
		web.Error(w, http.StatusBadRequest, "Saldo da conta não pode ser negativo")
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

	accounts, err := h.bankAccountRepo.ListByUserID(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao buscar contas bancárias")
		return
	}

	if len(accounts) == 0 {
		web.Error(w, http.StatusNotFound, "Nenhuma conta bancária encontrada para o usuário")
		return
	}

	targetAccount := accounts[0]
	if req.AccountID != nil && *req.AccountID != uuid.Nil {
		found := false
		for _, acc := range accounts {
			if acc.ID == *req.AccountID {
				targetAccount = acc
				found = true
				break
			}
		}
		if !found {
			web.Error(w, http.StatusNotFound, "Conta bancária especificada não encontrada")
			return
		}
	}

	previousBalance := targetAccount.CurrentBalance

	updatedAccount, err := h.bankAccountRepo.Update(ctx, targetAccount.ID, u.ID, targetAccount.Name, req.NewBalance)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao atualizar saldo da conta bancária")
		return
	}

	totalLiquid, err := h.userRepo.CalculateTotalLiquidBalance(ctx, u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao recalcular saldo total de liquidez")
		return
	}

	resp := AdjustBalanceResponse{
		AccountID:          updatedAccount.ID,
		AccountName:        updatedAccount.Name,
		PreviousBalance:    previousBalance,
		NewBalance:         updatedAccount.CurrentBalance,
		TotalLiquidBalance: totalLiquid,
		Message:            "Saldo bancário atualizado com sucesso",
	}

	web.JSON(w, http.StatusOK, resp)
}
