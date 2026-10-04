package investment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
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
