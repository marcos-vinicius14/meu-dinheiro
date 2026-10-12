package bankaccount

import (
	"encoding/binary"
	"errors"
	"fmt"
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

// CurrentBalanceSQL gera a expressão SQL canônica para calcular o saldo líquido atual de uma conta,
// somando o initial_balance com as transações confirmadas (INCOME positivo, EXPENSE negativo).
func CurrentBalanceSQL(initialBalanceCol, accountIDCol string) string {
	return fmt.Sprintf(`%s + COALESCE(
		(SELECT SUM(CASE 
			WHEN t.type = 'INCOME' AND t.status = 'CONFIRMED' THEN t.value 
			WHEN t.type IN ('FIXED_EXPENSE', 'FLEXIBLE_EXPENSE', 'INSTALLMENT_EXPENSE') AND t.status = 'CONFIRMED' THEN -t.value 
			ELSE 0 END) 
		 FROM tb_transactions t 
		 WHERE t.bank_account_id = %s), 0
	)`, initialBalanceCol, accountIDCol)
}
