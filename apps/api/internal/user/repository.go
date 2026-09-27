package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
)

var ErrUserNotFound = errors.New("usuário não encontrado")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, created_at
		FROM tb_users
		WHERE id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.FirstName,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("buscar usuário por id: %w", err)
	}
	return &u, nil
}

func (r *Repository) FindByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, created_at
		FROM tb_users
		WHERE telegram_id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.FirstName,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("buscar usuário por telegram_id: %w", err)
	}
	return &u, nil
}

// FindOrCreateByTelegram busca o usuário pelo telegram_id; se não existir, cria e provisiona dados iniciais.
func (r *Repository) FindOrCreateByTelegram(ctx context.Context, telegramID int64, username *string, firstName string) (*User, error) {
	existing, err := r.FindByTelegramID(ctx, telegramID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	var newUser User
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Insere o usuário
		insertUserQuery := `
			INSERT INTO tb_users (telegram_id, username, first_name)
			VALUES ($1, $2, $3)
			RETURNING id, telegram_id, username, first_name, created_at
		`
		if err := tx.QueryRow(ctx, insertUserQuery, telegramID, username, firstName).Scan(
			&newUser.ID,
			&newUser.TelegramID,
			&newUser.Username,
			&newUser.FirstName,
			&newUser.CreatedAt,
		); err != nil {
			return fmt.Errorf("inserir novo usuário: %w", err)
		}

		// Provisiona categorias padrão
		defaultCategories := []struct {
			desc     string
			icon     string
			flexible bool
		}{
			{"Alimentação", "utensils", true},
			{"Moradia", "home", false},
			{"Transporte", "car", true},
			{"Lazer", "gamepad", true},
			{"Salário", "wallet", false},
		}

		for _, cat := range defaultCategories {
			_, err := tx.Exec(ctx, `
				INSERT INTO tb_categories (user_id, description, icon, is_flexible)
				VALUES ($1, $2, $3, $4)
			`, newUser.ID, cat.desc, cat.icon, cat.flexible)
			if err != nil {
				return fmt.Errorf("provisionar categoria %s: %w", cat.desc, err)
			}
		}

		// Provisiona conta bancária padrão
		_, err := tx.Exec(ctx, `
			INSERT INTO tb_bank_accounts (user_id, name, initial_balance, type)
			VALUES ($1, 'Conta Corrente', 0.00, 'CHECKING')
		`, newUser.ID)
		if err != nil {
			return fmt.Errorf("provisionar conta bancária padrão: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &newUser, nil
}
