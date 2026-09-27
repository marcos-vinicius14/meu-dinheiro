package bankaccount

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, name string, accountType string, initialBalance money.Money) (*BankAccount, error) {
	name = strings.TrimSpace(name)
	if !ValidateName(name) {
		return nil, ErrInvalidAccountName
	}
	if !ValidateType(accountType) {
		return nil, ErrInvalidAccountType
	}

	var account BankAccount
	lockID := UserAdvisoryLockID(userID)

	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Adquire advisory lock transacional por usuário
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
			return fmt.Errorf("adquirir lock de criação: %w", err)
		}

		var count int
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM tb_bank_accounts WHERE user_id = $1", userID).Scan(&count); err != nil {
			return fmt.Errorf("contar contas do usuário: %w", err)
		}

		if count >= 3 {
			return ErrAccountLimitReached
		}

		query := `
			INSERT INTO tb_bank_accounts (user_id, name, initial_balance, type)
			VALUES ($1, $2, $3, $4)
			RETURNING id, user_id, name, type, initial_balance
		`
		return tx.QueryRow(ctx, query, userID, name, initialBalance, strings.ToUpper(accountType)).Scan(
			&account.ID,
			&account.UserID,
			&account.Name,
			&account.Type,
			&account.InitialBalance,
		)
	})

	if err != nil {
		return nil, err
	}

	account.CurrentBalance = account.InitialBalance
	return &account, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]BankAccount, error) {
	query := `
		SELECT id, user_id, name, type, initial_balance
		FROM tb_bank_accounts
		WHERE user_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar contas: %w", err)
	}
	defer rows.Close()

	var accounts []BankAccount
	for rows.Next() {
		var a BankAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.InitialBalance); err != nil {
			return nil, fmt.Errorf("ler conta: %w", err)
		}
		a.CurrentBalance = a.InitialBalance
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []BankAccount{}
	}
	return accounts, nil
}

func (r *Repository) FindByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*BankAccount, error) {
	query := `
		SELECT id, user_id, name, type, initial_balance
		FROM tb_bank_accounts
		WHERE id = $1 AND user_id = $2
	`
	var a BankAccount
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.InitialBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("buscar conta: %w", err)
	}
	a.CurrentBalance = a.InitialBalance
	return &a, nil
}

func (r *Repository) Update(ctx context.Context, id, userID uuid.UUID, name string, initialBalance money.Money) (*BankAccount, error) {
	name = strings.TrimSpace(name)
	if !ValidateName(name) {
		return nil, ErrInvalidAccountName
	}

	query := `
		UPDATE tb_bank_accounts
		SET name = $1, initial_balance = $2
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, name, type, initial_balance
	`
	var a BankAccount
	err := r.pool.QueryRow(ctx, query, name, initialBalance, id, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.Name,
		&a.Type,
		&a.InitialBalance,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("atualizar conta: %w", err)
	}
	a.CurrentBalance = a.InitialBalance
	return &a, nil
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// 1. Verifica se a conta existe e pertence ao usuário
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tb_bank_accounts WHERE id = $1 AND user_id = $2)", id, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAccountNotFound
	}

	// 2. Verifica se existem transações vinculadas à conta
	var hasTx bool
	err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tb_transactions WHERE bank_account_id = $1)", id).Scan(&hasTx)
	if err != nil {
		return err
	}
	if hasTx {
		return ErrAccountHasTransactions
	}

	// 3. Deleta a conta
	_, err = r.pool.Exec(ctx, "DELETE FROM tb_bank_accounts WHERE id = $1 AND user_id = $2", id, userID)
	return err
}
