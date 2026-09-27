package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   *string   `json:"username,omitempty"`
	FirstName  string    `json:"first_name"`
	CreatedAt  time.Time `json:"created_at"`
}
