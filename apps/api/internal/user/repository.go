package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
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
		SELECT id, telegram_id, username, first_name, target_savings,
		       flexible_budget_cap, emergency_fund_target, emergency_fund_months,
		       cycle_start_day, created_at
		FROM tb_users
		WHERE id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.FirstName,
		&u.TargetSavings,
		&u.FlexibleBudgetCap,
		&u.EmergencyFundTarget,
		&u.EmergencyFundMonths,
		&u.CycleStartDay,
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
		SELECT id, telegram_id, username, first_name, target_savings,
		       flexible_budget_cap, emergency_fund_target, emergency_fund_months,
		       cycle_start_day, created_at
		FROM tb_users
		WHERE telegram_id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, telegramID).Scan(
		&u.ID,
		&u.TelegramID,
		&u.Username,
		&u.FirstName,
		&u.TargetSavings,
		&u.FlexibleBudgetCap,
		&u.EmergencyFundTarget,
		&u.EmergencyFundMonths,
		&u.CycleStartDay,
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
			RETURNING id, telegram_id, username, first_name, target_savings,
			          flexible_budget_cap, emergency_fund_target, emergency_fund_months,
			          cycle_start_day, created_at
		`
		if err := tx.QueryRow(ctx, insertUserQuery, telegramID, username, firstName).Scan(
			&newUser.ID,
			&newUser.TelegramID,
			&newUser.Username,
			&newUser.FirstName,
			&newUser.TargetSavings,
			&newUser.FlexibleBudgetCap,
			&newUser.EmergencyFundTarget,
			&newUser.EmergencyFundMonths,
			&newUser.CycleStartDay,
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
			{"Saúde", "heart", false},
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

// UpdateFinancialProfile atualiza as metas financeiras, reserva e dia de início do ciclo.
func (r *Repository) UpdateFinancialProfile(
	ctx context.Context,
	userID uuid.UUID,
	targetSavings, flexibleBudgetCap, emergencyTarget money.Money,
	emergencyMonths, cycleStartDay int,
) error {
	if cycleStartDay < 1 || cycleStartDay > 28 {
		cycleStartDay = 1
	}
	if emergencyMonths != 6 && emergencyMonths != 12 {
		emergencyMonths = 6
	}

	query := `
		UPDATE tb_users
		SET target_savings = $1,
		    flexible_budget_cap = $2,
		    emergency_fund_target = $3,
		    emergency_fund_months = $4,
		    cycle_start_day = $5
		WHERE id = $6
	`
	result, err := r.pool.Exec(ctx, query,
		targetSavings, flexibleBudgetCap, emergencyTarget,
		emergencyMonths, cycleStartDay, userID,
	)
	if err != nil {
		return fmt.Errorf("atualizar perfil financeiro: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CalculateTotalLiquidBalance calcula a liquidez total real:
// soma de initial_balance das contas + (receitas confirmadas - despesas confirmadas)
func (r *Repository) CalculateTotalLiquidBalance(ctx context.Context, userID uuid.UUID) (money.Money, error) {
	query := `
		WITH account_balances AS (
			SELECT COALESCE(SUM(initial_balance), 0) AS total_initial
			FROM tb_bank_accounts
			WHERE user_id = $1
		),
		transaction_deltas AS (
			SELECT
				COALESCE(SUM(CASE WHEN type = 'INCOME' AND status = 'CONFIRMED' THEN value ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type IN ('FIXED_EXPENSE', 'FLEXIBLE_EXPENSE', 'INSTALLMENT_EXPENSE') AND status = 'CONFIRMED' THEN value ELSE 0 END), 0) AS net_transactions
			FROM tb_transactions
			WHERE user_id = $1
		)
		SELECT total_initial + net_transactions
		FROM account_balances, transaction_deltas
	`
	var balance money.Money
	err := r.pool.QueryRow(ctx, query, userID).Scan(&balance)
	if err != nil {
		return money.Zero(), fmt.Errorf("calcular liquidez total: %w", err)
	}
	return balance, nil
}

// CalculateMonthlyEssentialCost calcula o total mensal de gastos essenciais/fixos (FIXED_EXPENSE).
func (r *Repository) CalculateMonthlyEssentialCost(ctx context.Context, userID uuid.UUID) (money.Money, error) {
	query := `
		SELECT COALESCE(SUM(value), 0)
		FROM tb_transactions
		WHERE user_id = $1 AND type = 'FIXED_EXPENSE' AND status != 'CANCELED'
	`
	var cost money.Money
	err := r.pool.QueryRow(ctx, query, userID).Scan(&cost)
	if err != nil {
		return money.Zero(), fmt.Errorf("calcular custo essencial mensal: %w", err)
	}
	return cost, nil
}
