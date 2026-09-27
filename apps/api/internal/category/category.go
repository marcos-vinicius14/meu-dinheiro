package category

import (
	"encoding/binary"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrCategoryNotFound        = errors.New("categoria não encontrada")
	ErrCategoryDuplicated      = errors.New("categoria já cadastrada")
	ErrCategoryHasTransactions = errors.New("categoria possui transações vinculadas")
	ErrInvalidDescription      = errors.New("descrição da categoria deve ter entre 1 e 255 caracteres")
)

type Category struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"-"`
	Description string    `json:"description"`
	Icon        *string   `json:"icon"`
	IsFlexible  bool      `json:"is_flexible"`
}

func ValidateDescription(desc string) bool {
	trimmed := strings.TrimSpace(desc)
	return len(trimmed) >= 1 && len(trimmed) <= 255
}

// UserAdvisoryLockID gera chave de lock para criação de categorias.
func UserAdvisoryLockID(userID uuid.UUID) int64 {
	hi := binary.BigEndian.Uint64(userID[0:8])
	lo := binary.BigEndian.Uint64(userID[8:16])
	return int64(hi ^ lo)
}
