package botapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/shopspring/decimal"
)

type SimulationRequest struct {
	TelegramID   int64       `json:"telegram_id"`
	Amount       money.Money `json:"amount"`
	Installments int         `json:"installments,omitempty"`
	FirstDueDate string      `json:"first_due_date,omitempty"`
}

type SimulationResponse struct {
	TotalAmount                  money.Money                      `json:"total_amount"`
	Installments                 int                              `json:"installments"`
	InstallmentAmount            money.Money                      `json:"installment_amount"`
	CurrentCycleS2SBefore        money.Money                      `json:"current_cycle_s2s_before"`
	CurrentCycleS2SAfter         money.Money                      `json:"current_cycle_s2s_after"`
	CurrentCycleReduction        money.Money                      `json:"current_cycle_reduction"`
	CurrentCycleReductionPercent decimal.Decimal                  `json:"current_cycle_reduction_percent"`
	CriticalCycleNumber          int                              `json:"critical_cycle_number"`
	CriticalCycleS2S             money.Money                      `json:"critical_cycle_s2s"`
	CriticalCycleHealthStatus    engine.HealthStatus              `json:"critical_cycle_health_status"`
	DeficitRiskAlert             bool                             `json:"deficit_risk_alert"`
	RecommendationMessage        string                           `json:"recommendation_message"`
	Cycles                       []transaction.SimulationCycleDto `json:"cycles"`
}

func (h *Handler) handleSimulatePurchase(w http.ResponseWriter, r *http.Request) {
	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	if req.TelegramID == 0 {
		web.Error(w, http.StatusBadRequest, "telegram_id é obrigatório")
		return
	}

	if !req.Amount.IsPositive() {
		web.Error(w, http.StatusBadRequest, "Valor da simulação deve ser maior que zero")
		return
	}

	if req.Installments <= 0 {
		req.Installments = 1
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

	now := time.Now().UTC()
	firstDueDate := now
	if req.FirstDueDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.FirstDueDate); err == nil {
			firstDueDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
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

	input := transaction.SimulationInput{
		Date:              now,
		LiquidBalance:     totalLiquid,
		TargetSavings:     u.TargetSavings,
		FlexibleBudgetCap: flexCap,
		TotalAmount:       req.Amount,
		Installments:      req.Installments,
		FirstDueDate:      firstDueDate,
	}

	cycles, err := h.txService.SimulatePurchase(ctx, u.ID, input)
	if err != nil {
		web.Error(w, http.StatusBadRequest, fmt.Sprintf("Erro ao simular compra: %v", err))
		return
	}

	if len(cycles) == 0 {
		web.Error(w, http.StatusInternalServerError, "Nenhum ciclo gerado na simulação")
		return
	}

	// 1. Ciclo Atual
	currentCycle := cycles[0]
	currentS2SAfter := currentCycle.S2SToday
	currentS2SReduction := currentCycle.S2SReduction
	currentS2SBefore := currentS2SAfter.Add(currentS2SReduction)
	currentReductionPercent := currentCycle.S2SReductionPercent

	installmentAmount := money.New(req.Amount.Decimal().Div(decimal.NewFromInt(int64(req.Installments))))

	// 2. Identifica o ciclo mais crítico (bottleneck ou menor S2S projetado)
	criticalCycleIdx := 0
	lowestS2S := cycles[0].S2SToday

	for i, c := range cycles {
		if c.Bottleneck {
			criticalCycleIdx = i
			lowestS2S = c.S2SToday
			break
		}
		if c.S2SToday.IsLessThan(lowestS2S) {
			lowestS2S = c.S2SToday
			criticalCycleIdx = i
		}
	}

	criticalCycle := cycles[criticalCycleIdx]
	criticalCycleNumber := criticalCycleIdx + 1

	deficitAlert := criticalCycle.HealthStatus == engine.HealthDeficitRisk || criticalCycle.HealthStatus == engine.HealthRestricted
	var recommendationMsg string

	if criticalCycle.HealthStatus == engine.HealthDeficitRisk {
		recommendationMsg = fmt.Sprintf("Atenção: o Ciclo %d entrará em RISCO DE DÉFICIT. Considere aumentar as parcelas ou adiar a compra.", criticalCycleNumber)
	} else if criticalCycle.HealthStatus == engine.HealthRestricted {
		recommendationMsg = fmt.Sprintf("Alerta: o Ciclo %d ficará com orçamento RESTRITO. Planeje reduzir despesas flexíveis.", criticalCycleNumber)
	} else {
		recommendationMsg = "Compra viável e saudável: todos os ciclos projetados mantêm margem segura."
	}

	resp := SimulationResponse{
		TotalAmount:                  req.Amount,
		Installments:                 req.Installments,
		InstallmentAmount:            installmentAmount,
		CurrentCycleS2SBefore:        currentS2SBefore,
		CurrentCycleS2SAfter:         currentS2SAfter,
		CurrentCycleReduction:        currentS2SReduction,
		CurrentCycleReductionPercent: currentReductionPercent,
		CriticalCycleNumber:          criticalCycleNumber,
		CriticalCycleS2S:             criticalCycle.S2SToday,
		CriticalCycleHealthStatus:    criticalCycle.HealthStatus,
		DeficitRiskAlert:             deficitAlert,
		RecommendationMessage:        recommendationMsg,
		Cycles:                       cycles,
	}

	web.JSON(w, http.StatusOK, resp)
}
