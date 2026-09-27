package transaction

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
)

var (
	ErrTransactionNotFound      = errors.New("transação não encontrada")
	ErrBundleNotFound           = errors.New("parcelamento não encontrado")
	ErrAmountMustBePositive     = errors.New("valor da transação deve ser positivo")
	ErrDueDateRequired          = errors.New("data de vencimento é obrigatória")
	ErrTypeRequired             = errors.New("tipo de transação é obrigatório")
	ErrCategoryRequired         = errors.New("categoria é obrigatória")
	ErrInstallmentImmutable     = errors.New("parcelas de um parcelamento não podem ser alteradas individualmente")
	ErrInstallmentTypeResticted = errors.New("tipo INSTALLMENT_EXPENSE é gerenciado pelo bundle")
	ErrInstallmentsRequired     = errors.New("número de parcelas deve ser positivo")
	ErrFirstDueDateRequired     = errors.New("data do primeiro vencimento é obrigatória")
	ErrCheckInDateRequired      = errors.New("data do check-in é obrigatória")
	ErrSimulationDateRequired   = errors.New("data da simulação é obrigatória")
	ErrCanceledCannotConfirm    = errors.New("transação cancelada não pode ser confirmada")
)

type Transaction struct {
	ID                uuid.UUID                `json:"id"`
	UserID            uuid.UUID                `json:"-"`
	BankAccountID     *uuid.UUID               `json:"bank_account_id"`
	CategoryID        uuid.UUID                `json:"category_id"`
	Description       string                   `json:"description"`
	Amount            money.Money              `json:"amount"`
	Type              engine.TransactionType   `json:"type"`
	Status            engine.TransactionStatus `json:"status"`
	DueDate           time.Time                `json:"due_date"`
	PaymentDate       *time.Time               `json:"payment_date"`
	BundleID          *uuid.UUID               `json:"bundle_id"`
	InstallmentNumber *int                     `json:"installment_number"`
	TotalInstallments *int                     `json:"total_installments"`
}

type Bundle struct {
	ID                uuid.UUID   `json:"id"`
	UserID            uuid.UUID   `json:"-"`
	Description       string      `json:"description"`
	TotalAmount       money.Money `json:"total_amount"`
	TotalInstallments int         `json:"total_installments"`
	FirstDueDate      time.Time   `json:"first_due_date"`
}

type CheckInSnapshot struct {
	ID                   uuid.UUID           `json:"id"`
	UserID               uuid.UUID           `json:"-"`
	CheckInDate          time.Time           `json:"check_in_date"`
	S2SCalculated        money.Money         `json:"s2s_calculated"`
	SpentToday           money.Money         `json:"spent_today"`
	DeltaFromSafeToSpend money.Money         `json:"delta_from_safe_to_spend"`
	HealthStatus         engine.HealthStatus `json:"health_status"`
	CreatedAt            time.Time           `json:"created_at"`
}
