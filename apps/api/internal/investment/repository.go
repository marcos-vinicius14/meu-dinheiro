package investment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/shopspring/decimal"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// AddLotResult representa o resultado de um aporte que debitou fundos da conta bancária.
type AddLotResult struct {
	Investment     *Investment
	DebitedAmount  money.Money
	AccountID      uuid.UUID
	AccountName    string
	AccountBalance money.Money
	TransactionID  uuid.UUID
}

// AddLotWithDebit registra o lote de investimento e debita o saldo da conta bancária de forma atômica (WithTx).
func (r *Repository) AddLotWithDebit(
	ctx context.Context,
	userID uuid.UUID,
	accountID *uuid.UUID,
	ticker string,
	quantity decimal.Decimal,
	price money.Money,
) (*AddLotResult, error) {
	normTicker, err := NormalizeTicker(ticker)
	if err != nil {
		return nil, err
	}
	if !quantity.IsPositive() {
		return nil, ErrInvalidQuantity
	}
	if price.IsNegative() {
		return nil, ErrInvalidPrice
	}

	totalDebit := CalculateTotalCost(quantity, price)

	var result AddLotResult
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// 1. Lock defensivo do usuário para prevenir concorrência em saldo e limites
		lockID := bankaccount.UserAdvisoryLockID(userID)
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
			return fmt.Errorf("adquirir lock defensivo: %w", err)
		}

		// 2. Localiza a conta bancária alvo com lock de linha (FOR UPDATE)
		var targetAccount bankaccount.BankAccount
		if accountID != nil && *accountID != uuid.Nil {
			accountQuery := `
				SELECT id, user_id, name, type, initial_balance
				FROM tb_bank_accounts
				WHERE id = $1 AND user_id = $2
				FOR UPDATE
			`
			err := tx.QueryRow(ctx, accountQuery, *accountID, userID).Scan(
				&targetAccount.ID,
				&targetAccount.UserID,
				&targetAccount.Name,
				&targetAccount.Type,
				&targetAccount.InitialBalance,
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return bankaccount.ErrAccountNotFound
				}
				return fmt.Errorf("buscar conta bancária informada: %w", err)
			}
		} else {
			accountQuery := `
				SELECT id, user_id, name, type, initial_balance
				FROM tb_bank_accounts
				WHERE user_id = $1
				ORDER BY CASE WHEN type = 'CHECKING' THEN 0 ELSE 1 END, name ASC
				LIMIT 1
				FOR UPDATE
			`
			err := tx.QueryRow(ctx, accountQuery, userID).Scan(
				&targetAccount.ID,
				&targetAccount.UserID,
				&targetAccount.Name,
				&targetAccount.Type,
				&targetAccount.InitialBalance,
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrNoAccountAvailable
				}
				return fmt.Errorf("buscar conta bancária padrão: %w", err)
			}
		}

		// 3. Calcula saldo atual da conta dentro da transação usando expressão canônica
		balanceQuery := fmt.Sprintf("SELECT %s", bankaccount.CurrentBalanceSQL("$1", "$2"))
		var currentBalance money.Money
		err = tx.QueryRow(ctx, balanceQuery, targetAccount.InitialBalance, targetAccount.ID).Scan(&currentBalance)
		if err != nil {
			return fmt.Errorf("calcular saldo da conta: %w", err)
		}

		// 4. Valida se o saldo da conta é suficiente
		if currentBalance.IsLessThan(totalDebit) {
			return ErrInsufficientBalance
		}

		// 5. Atualiza ou insere a custódia do investimento
		selectQuery := `
			SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
			FROM tb_investments
			WHERE user_id = $1 AND ticker = $2
			FOR UPDATE
		`
		var existing Investment
		err = tx.QueryRow(ctx, selectQuery, userID, normTicker).Scan(
			&existing.ID,
			&existing.UserID,
			&existing.Ticker,
			&existing.Quantity,
			&existing.AveragePrice,
			&existing.CreatedAt,
			&existing.UpdatedAt,
		)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("buscar investimento existente: %w", err)
		}

		var inv Investment
		if errors.Is(err, pgx.ErrNoRows) {
			newQty, newPM, err := CalculateNewAveragePrice(decimal.Zero, money.Zero(), quantity, price)
			if err != nil {
				return err
			}
			insertQuery := `
				INSERT INTO tb_investments (user_id, ticker, quantity, average_price)
				VALUES ($1, $2, $3, $4)
				RETURNING id, user_id, ticker, quantity, average_price, created_at, updated_at
			`
			if err := tx.QueryRow(ctx, insertQuery, userID, normTicker, newQty, newPM).Scan(
				&inv.ID,
				&inv.UserID,
				&inv.Ticker,
				&inv.Quantity,
				&inv.AveragePrice,
				&inv.CreatedAt,
				&inv.UpdatedAt,
			); err != nil {
				return fmt.Errorf("inserir novo investimento: %w", err)
			}
		} else {
			newQty, newPM, err := CalculateNewAveragePrice(existing.Quantity, existing.AveragePrice, quantity, price)
			if err != nil {
				return err
			}
			updateQuery := `
				UPDATE tb_investments
				SET quantity = $1, average_price = $2, updated_at = CURRENT_TIMESTAMP
				WHERE id = $3
				RETURNING id, user_id, ticker, quantity, average_price, created_at, updated_at
			`
			if err := tx.QueryRow(ctx, updateQuery, newQty, newPM, existing.ID).Scan(
				&inv.ID,
				&inv.UserID,
				&inv.Ticker,
				&inv.Quantity,
				&inv.AveragePrice,
				&inv.CreatedAt,
				&inv.UpdatedAt,
			); err != nil {
				return fmt.Errorf("atualizar investimento existente: %w", err)
			}
		}
		inv.TotalCost = CalculateTotalCost(inv.Quantity, inv.AveragePrice)

		// 6. Garante a categoria "Investimentos"
		var categoryID uuid.UUID
		catQuery := `
			SELECT id
			FROM tb_categories
			WHERE user_id = $1 AND LOWER(description) = 'investimentos'
			LIMIT 1
		`
		err = tx.QueryRow(ctx, catQuery, userID).Scan(&categoryID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				insertCatQuery := `
					INSERT INTO tb_categories (user_id, description, icon, is_flexible)
					VALUES ($1, 'Investimentos', 'trending-up', true)
					RETURNING id
				`
				if err := tx.QueryRow(ctx, insertCatQuery, userID).Scan(&categoryID); err != nil {
					return fmt.Errorf("criar categoria investimentos: %w", err)
				}
			} else {
				return fmt.Errorf("buscar categoria investimentos: %w", err)
			}
		}

		// 7. Insere transação de débito correspondente
		now := time.Now().UTC()
		var transactionID uuid.UUID
		insertTxQuery := `
			INSERT INTO tb_transactions (
				user_id, bank_account_id, category_id, description,
				value, type, status, due_date, payment_date
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`
		txDesc := "Aporte " + normTicker
		if err := tx.QueryRow(
			ctx,
			insertTxQuery,
			userID,
			targetAccount.ID,
			categoryID,
			txDesc,
			totalDebit,
			engine.TypeFlexibleExpense,
			engine.StatusConfirmed,
			now,
			now,
		).Scan(&transactionID); err != nil {
			return fmt.Errorf("inserir transação de débito do aporte: %w", err)
		}

		result = AddLotResult{
			Investment:     &inv,
			DebitedAmount:  totalDebit,
			AccountID:      targetAccount.ID,
			AccountName:    targetAccount.Name,
			AccountBalance: currentBalance.Subtract(totalDebit),
			TransactionID:  transactionID,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// AddLot adiciona um novo lote de um ativo, recalculando atomicamente o Preço Médio Ponderado.
func (r *Repository) AddLot(
	ctx context.Context,
	userID uuid.UUID,
	ticker string,
	quantity decimal.Decimal,
	price money.Money,
) (*Investment, error) {
	normTicker, err := NormalizeTicker(ticker)
	if err != nil {
		return nil, err
	}

	var inv Investment
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		selectQuery := `
			SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
			FROM tb_investments
			WHERE user_id = $1 AND ticker = $2
			FOR UPDATE
		`
		var existing Investment
		err := tx.QueryRow(ctx, selectQuery, userID, normTicker).Scan(
			&existing.ID,
			&existing.UserID,
			&existing.Ticker,
			&existing.Quantity,
			&existing.AveragePrice,
			&existing.CreatedAt,
			&existing.UpdatedAt,
		)

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("buscar investimento existente: %w", err)
		}

		if errors.Is(err, pgx.ErrNoRows) {
			// Inserção de novo ativo
			newQty, newPM, err := CalculateNewAveragePrice(decimal.Zero, money.Zero(), quantity, price)
			if err != nil {
				return err
			}

			insertQuery := `
				INSERT INTO tb_investments (user_id, ticker, quantity, average_price)
				VALUES ($1, $2, $3, $4)
				RETURNING id, user_id, ticker, quantity, average_price, created_at, updated_at
			`
			return tx.QueryRow(ctx, insertQuery, userID, normTicker, newQty, newPM).Scan(
				&inv.ID,
				&inv.UserID,
				&inv.Ticker,
				&inv.Quantity,
				&inv.AveragePrice,
				&inv.CreatedAt,
				&inv.UpdatedAt,
			)
		}

		// Recálculo de Preço Médio Ponderado para posição existente
		newQty, newPM, err := CalculateNewAveragePrice(existing.Quantity, existing.AveragePrice, quantity, price)
		if err != nil {
			return err
		}

		updateQuery := `
			UPDATE tb_investments
			SET quantity = $1, average_price = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
			RETURNING id, user_id, ticker, quantity, average_price, created_at, updated_at
		`
		return tx.QueryRow(ctx, updateQuery, newQty, newPM, existing.ID).Scan(
			&inv.ID,
			&inv.UserID,
			&inv.Ticker,
			&inv.Quantity,
			&inv.AveragePrice,
			&inv.CreatedAt,
			&inv.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	inv.TotalCost = CalculateTotalCost(inv.Quantity, inv.AveragePrice)
	return &inv, nil
}

// Sell abate quantidade de um ativo da carteira mantendo o Preço Médio inalterado. Se zerar, deleta o registro.
func (r *Repository) Sell(
	ctx context.Context,
	userID uuid.UUID,
	ticker string,
	quantity decimal.Decimal,
) (*Investment, error) {
	normTicker, err := NormalizeTicker(ticker)
	if err != nil {
		return nil, err
	}

	var inv Investment
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		selectQuery := `
			SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
			FROM tb_investments
			WHERE user_id = $1 AND ticker = $2
			FOR UPDATE
		`
		var existing Investment
		err := tx.QueryRow(ctx, selectQuery, userID, normTicker).Scan(
			&existing.ID,
			&existing.UserID,
			&existing.Ticker,
			&existing.Quantity,
			&existing.AveragePrice,
			&existing.CreatedAt,
			&existing.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvestmentNotFound
			}
			return fmt.Errorf("buscar investimento para venda: %w", err)
		}

		remainingQty, isClosed, err := CalculateSale(existing.Quantity, quantity)
		if err != nil {
			return err
		}

		if isClosed {
			_, err := tx.Exec(ctx, "DELETE FROM tb_investments WHERE id = $1", existing.ID)
			if err != nil {
				return fmt.Errorf("deletar posicao zerada: %w", err)
			}
			inv = existing
			inv.Quantity = decimal.Zero
			inv.TotalCost = money.Zero()
			return nil
		}

		updateQuery := `
			UPDATE tb_investments
			SET quantity = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
			RETURNING id, user_id, ticker, quantity, average_price, created_at, updated_at
		`
		return tx.QueryRow(ctx, updateQuery, remainingQty, existing.ID).Scan(
			&inv.ID,
			&inv.UserID,
			&inv.Ticker,
			&inv.Quantity,
			&inv.AveragePrice,
			&inv.CreatedAt,
			&inv.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	inv.TotalCost = CalculateTotalCost(inv.Quantity, inv.AveragePrice)
	return &inv, nil
}

// ListByUserID lista todos os ativos do usuário ordenados por ticker alfabeticamente.
func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Investment, error) {
	query := `
		SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
		FROM tb_investments
		WHERE user_id = $1
		ORDER BY ticker ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar investimentos: %w", err)
	}
	defer rows.Close()

	var list []Investment
	for rows.Next() {
		var item Investment
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Ticker,
			&item.Quantity,
			&item.AveragePrice,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ler investimento: %w", err)
		}
		item.TotalCost = CalculateTotalCost(item.Quantity, item.AveragePrice)
		list = append(list, item)
	}

	if list == nil {
		list = []Investment{}
	}
	return list, nil
}

// FindByIDAndUserID busca um investimento por id e usuário.
func (r *Repository) FindByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*Investment, error) {
	query := `
		SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
		FROM tb_investments
		WHERE id = $1 AND user_id = $2
	`
	var inv Investment
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&inv.ID,
		&inv.UserID,
		&inv.Ticker,
		&inv.Quantity,
		&inv.AveragePrice,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvestmentNotFound
		}
		return nil, fmt.Errorf("buscar investimento por id: %w", err)
	}

	inv.TotalCost = CalculateTotalCost(inv.Quantity, inv.AveragePrice)
	return &inv, nil
}

// FindByTickerAndUserID busca um investimento pelo ticker e usuário.
func (r *Repository) FindByTickerAndUserID(ctx context.Context, ticker string, userID uuid.UUID) (*Investment, error) {
	normTicker, err := NormalizeTicker(ticker)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, ticker, quantity, average_price, created_at, updated_at
		FROM tb_investments
		WHERE user_id = $1 AND ticker = $2
	`
	var inv Investment
	err = r.pool.QueryRow(ctx, query, userID, normTicker).Scan(
		&inv.ID,
		&inv.UserID,
		&inv.Ticker,
		&inv.Quantity,
		&inv.AveragePrice,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvestmentNotFound
		}
		return nil, fmt.Errorf("buscar investimento por ticker: %w", err)
	}

	inv.TotalCost = CalculateTotalCost(inv.Quantity, inv.AveragePrice)
	return &inv, nil
}

// Delete remove um ativo da carteira do usuário.
func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM tb_investments WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("deletar investimento: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInvestmentNotFound
	}
	return nil
}

// CalculateTotalInvested calcula a soma total investida em ativos do usuário.
func (r *Repository) CalculateTotalInvested(ctx context.Context, userID uuid.UUID) (money.Money, error) {
	query := `
		SELECT COALESCE(SUM(quantity * average_price), 0)
		FROM tb_investments
		WHERE user_id = $1
	`
	var total money.Money
	err := r.pool.QueryRow(ctx, query, userID).Scan(&total)
	if err != nil {
		return money.Zero(), fmt.Errorf("calcular total investido: %w", err)
	}
	return total, nil
}
