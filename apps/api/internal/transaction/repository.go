package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, t *Transaction) (*Transaction, error) {
	dueDate := dateinterval.NormalizeDate(t.DueDate)
	var paymentDate *time.Time
	if t.PaymentDate != nil {
		pd := dateinterval.NormalizeDate(*t.PaymentDate)
		paymentDate = &pd
	}

	query := `
		INSERT INTO tb_transactions (
			user_id, bank_account_id, category_id, description,
			value, type, status, due_date, payment_date,
			bundle_id, installment_number, total_installments
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		t.UserID, t.BankAccountID, t.CategoryID, t.Description,
		t.Amount, t.Type, t.Status, dueDate, paymentDate,
		t.BundleID, t.InstallmentNumber, t.TotalInstallments,
	).Scan(&t.ID)

	if err != nil {
		return nil, fmt.Errorf("inserir transação: %w", err)
	}
	t.DueDate = dueDate
	t.PaymentDate = paymentDate
	return t, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Transaction, error) {
	query := `
		SELECT id, user_id, bank_account_id, category_id, description,
		       value, type, status, due_date, payment_date,
		       bundle_id, installment_number, total_installments
		FROM tb_transactions
		WHERE user_id = $1
		ORDER BY due_date DESC, id DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar transações: %w", err)
	}
	defer rows.Close()

	var list []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.BankAccountID, &t.CategoryID, &t.Description,
			&t.Amount, &t.Type, &t.Status, &t.DueDate, &t.PaymentDate,
			&t.BundleID, &t.InstallmentNumber, &t.TotalInstallments,
		); err != nil {
			return nil, fmt.Errorf("ler transação: %w", err)
		}
		list = append(list, t)
	}
	if list == nil {
		list = []Transaction{}
	}
	return list, nil
}

func (r *Repository) FindByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*Transaction, error) {
	query := `
		SELECT id, user_id, bank_account_id, category_id, description,
		       value, type, status, due_date, payment_date,
		       bundle_id, installment_number, total_installments
		FROM tb_transactions
		WHERE id = $1 AND user_id = $2
	`
	var t Transaction
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&t.ID, &t.UserID, &t.BankAccountID, &t.CategoryID, &t.Description,
		&t.Amount, &t.Type, &t.Status, &t.DueDate, &t.PaymentDate,
		&t.BundleID, &t.InstallmentNumber, &t.TotalInstallments,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("buscar transação: %w", err)
	}
	return &t, nil
}

func (r *Repository) Update(ctx context.Context, t *Transaction) error {
	dueDate := dateinterval.NormalizeDate(t.DueDate)
	query := `
		UPDATE tb_transactions
		SET description = $1, value = $2, type = $3, due_date = $4,
		    category_id = $5, bank_account_id = $6
		WHERE id = $7 AND user_id = $8
	`
	result, err := r.pool.Exec(ctx, query,
		t.Description, t.Amount, t.Type, dueDate,
		t.CategoryID, t.BankAccountID, t.ID, t.UserID,
	)
	if err != nil {
		return fmt.Errorf("atualizar transação: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTransactionNotFound
	}
	t.DueDate = dueDate
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM tb_transactions WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("deletar transação: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

func (r *Repository) CreateBundleWithInstallments(ctx context.Context, b *Bundle, installments []Transaction) (*Bundle, []Transaction, error) {
	firstDueDate := dateinterval.NormalizeDate(b.FirstDueDate)

	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		insertBundleQuery := `
			INSERT INTO tb_transaction_bundles (user_id, description, total_amount, total_installments, first_due_date)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		if err := tx.QueryRow(ctx, insertBundleQuery, b.UserID, b.Description, b.TotalAmount, b.TotalInstallments, firstDueDate).Scan(&b.ID); err != nil {
			return fmt.Errorf("inserir bundle: %w", err)
		}
		b.FirstDueDate = firstDueDate

		insertTxQuery := `
			INSERT INTO tb_transactions (
				user_id, bank_account_id, category_id, description,
				value, type, status, due_date, bundle_id,
				installment_number, total_installments
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`
		for i := range installments {
			inst := &installments[i]
			inst.BundleID = &b.ID
			instDueDate := dateinterval.NormalizeDate(inst.DueDate)
			if err := tx.QueryRow(ctx, insertTxQuery,
				inst.UserID, inst.BankAccountID, inst.CategoryID, inst.Description,
				inst.Amount, inst.Type, inst.Status, instDueDate, inst.BundleID,
				inst.InstallmentNumber, inst.TotalInstallments,
			).Scan(&inst.ID); err != nil {
				return fmt.Errorf("inserir parcela %d do bundle: %w", i+1, err)
			}
			inst.DueDate = instDueDate
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return b, installments, nil
}

func (r *Repository) ExecuteCheckInWrites(ctx context.Context, userID uuid.UUID, newExpenses []Transaction, confirmedIDs []uuid.UUID, confirmDate time.Time) error {
	normDate := dateinterval.NormalizeDate(confirmDate)

	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Insere despesas não rastreadas
		insertExpenseQuery := `
			INSERT INTO tb_transactions (
				user_id, category_id, description, value,
				type, status, due_date, payment_date
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		for _, exp := range newExpenses {
			_, err := tx.Exec(ctx, insertExpenseQuery,
				userID, exp.CategoryID, exp.Description, exp.Amount,
				engine.TypeFlexibleExpense, engine.StatusConfirmed, normDate, normDate,
			)
			if err != nil {
				return fmt.Errorf("inserir despesa do checkin: %w", err)
			}
		}

		// Confirma transações pendentes
		if len(confirmedIDs) > 0 {
			updatePendingQuery := `
				UPDATE tb_transactions
				SET status = 'CONFIRMED', payment_date = $1
				WHERE id = ANY($2) AND user_id = $3
			`
			if _, err := tx.Exec(ctx, updatePendingQuery, normDate, confirmedIDs, userID); err != nil {
				return fmt.Errorf("confirmar transações pendentes: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) UpsertSnapshot(ctx context.Context, s *CheckInSnapshot) error {
	normDate := dateinterval.NormalizeDate(s.CheckInDate)
	query := `
		INSERT INTO tb_check_in_snapshots (
			user_id, check_in_date, s2s_calculated, spent_today,
			delta_from_safe_to_spend, health_status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, check_in_date)
		DO UPDATE SET
			s2s_calculated = EXCLUDED.s2s_calculated,
			spent_today = EXCLUDED.spent_today,
			delta_from_safe_to_spend = EXCLUDED.delta_from_safe_to_spend,
			health_status = EXCLUDED.health_status
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		s.UserID, normDate, s.S2SCalculated, s.SpentToday,
		s.DeltaFromSafeToSpend, s.HealthStatus,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}
	s.CheckInDate = normDate
	return nil
}

func (r *Repository) LoadSnapshotsByUserID(ctx context.Context, userID uuid.UUID) ([]engine.TransactionSnapshot, error) {
	query := `
		SELECT type, status, value, due_date
		FROM tb_transactions
		WHERE user_id = $1
		ORDER BY due_date DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("carregar snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []engine.TransactionSnapshot
	for rows.Next() {
		var s engine.TransactionSnapshot
		if err := rows.Scan(&s.Type, &s.Status, &s.Amount, &s.DueDate); err != nil {
			return nil, fmt.Errorf("ler snapshot: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if snapshots == nil {
		snapshots = []engine.TransactionSnapshot{}
	}
	return snapshots, nil
}
