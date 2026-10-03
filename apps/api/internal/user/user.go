package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
)

type User struct {
	ID                  uuid.UUID   `json:"id"`
	TelegramID          int64       `json:"telegram_id"`
	Username            *string     `json:"username,omitempty"`
	FirstName           string      `json:"first_name"`
	TargetSavings       money.Money `json:"target_savings"`
	FlexibleBudgetCap   money.Money `json:"flexible_budget_cap"`
	EmergencyFundTarget money.Money `json:"emergency_fund_target"`
	EmergencyFundMonths int         `json:"emergency_fund_months"`
	CycleStartDay       int         `json:"cycle_start_day"`
	CreatedAt           time.Time   `json:"created_at"`
}
