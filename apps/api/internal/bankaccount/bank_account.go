package bankaccount

import (
	"encoding/binary"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
)

var (
	ErrAccountNotFound        = errors.New("conta bancária não encontrada")
	ErrAccountLimitReached    = errors.New("limite de 3 contas bancárias atingido")
	ErrAccountHasTransactions = errors.New("conta bancária possui transações vinculadas")
	ErrInvalidAccountType     = errors.New("tipo de conta inválido. Tipos aceitos: CHECKING, SAVINGS, INVESTMENT")
	ErrInvalidAccountName     = errors.New("nome da conta bancária deve ter entre 1 e 255 caracteres")
)

type BankAccount struct {
	ID             uuid.UUID   `json:"id"`
	UserID         uuid.UUID   `json:"-"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	InitialBalance money.Money `json:"initial_balance"`
	CurrentBalance money.Money `json:"current_balance"`
}

func ValidateType(accountType string) bool {
	switch strings.ToUpper(accountType) {
	case "CHECKING", "SAVINGS", "INVESTMENT":
		return true
	default:
		return false
	}
}

func ValidateName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return len(trimmed) >= 1 && len(trimmed) <= 255
}

// UserAdvisoryLockID gera uma chave int64 para pg_advisory_xact_lock a partir do UUID.
func UserAdvisoryLockID(userID uuid.UUID) int64 {
	hi := binary.BigEndian.Uint64(userID[0:8])
	lo := binary.BigEndian.Uint64(userID[8:16])
	return int64(hi ^ lo)
}
