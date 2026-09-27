package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, description string, icon *string, isFlexible bool) (*Category, error) {
	description = strings.TrimSpace(description)
	if !ValidateDescription(description) {
		return nil, ErrInvalidDescription
	}

	var c Category
	lockID := UserAdvisoryLockID(userID)

	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Advisory lock para serializar criações concorrentes do mesmo usuário
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
			return fmt.Errorf("adquirir lock de criação: %w", err)
		}

		var exists bool
		checkQuery := `
			SELECT EXISTS(
				SELECT 1 FROM tb_categories
				WHERE user_id = $1 AND LOWER(description) = LOWER($2)
			)
		`
		if err := tx.QueryRow(ctx, checkQuery, userID, description).Scan(&exists); err != nil {
			return fmt.Errorf("verificar duplicidade: %w", err)
		}
		if exists {
			return ErrCategoryDuplicated
		}

		insertQuery := `
			INSERT INTO tb_categories (user_id, description, icon, is_flexible)
			VALUES ($1, $2, $3, $4)
			RETURNING id, user_id, description, icon, is_flexible
		`
		return tx.QueryRow(ctx, insertQuery, userID, description, icon, isFlexible).Scan(
			&c.ID,
			&c.UserID,
			&c.Description,
			&c.Icon,
			&c.IsFlexible,
		)
	})

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Category, error) {
	query := `
		SELECT id, user_id, description, icon, is_flexible
		FROM tb_categories
		WHERE user_id = $1
		ORDER BY description ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar categorias: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.Description, &c.Icon, &c.IsFlexible); err != nil {
			return nil, fmt.Errorf("ler categoria: %w", err)
		}
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []Category{}
	}
	return categories, nil
}

func (r *Repository) FindByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*Category, error) {
	query := `
		SELECT id, user_id, description, icon, is_flexible
		FROM tb_categories
		WHERE id = $1 AND user_id = $2
	`
	var c Category
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(&c.ID, &c.UserID, &c.Description, &c.Icon, &c.IsFlexible)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("buscar categoria: %w", err)
	}
	return &c, nil
}

func (r *Repository) Update(ctx context.Context, id, userID uuid.UUID, description string, icon *string) (*Category, error) {
	description = strings.TrimSpace(description)
	if !ValidateDescription(description) {
		return nil, ErrInvalidDescription
	}

	// Verifica se outra categoria do mesmo usuário já tem essa descrição
	var duplicate bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM tb_categories
			WHERE user_id = $1 AND LOWER(description) = LOWER($2) AND id <> $3
		)
	`
	if err := r.pool.QueryRow(ctx, checkQuery, userID, description, id).Scan(&duplicate); err != nil {
		return nil, fmt.Errorf("checar duplicidade na alteração: %w", err)
	}
	if duplicate {
		return nil, ErrCategoryDuplicated
	}

	query := `
		UPDATE tb_categories
		SET description = $1, icon = $2
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, description, icon, is_flexible
	`
	var c Category
	err := r.pool.QueryRow(ctx, query, description, icon, id, userID).Scan(
		&c.ID,
		&c.UserID,
		&c.Description,
		&c.Icon,
		&c.IsFlexible,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("atualizar categoria: %w", err)
	}
	return &c, nil
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// Deleção atômica condicional: só deleta se não houver transações vinculadas
	deleteQuery := `
		DELETE FROM tb_categories
		WHERE id = $1 AND user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM tb_transactions WHERE category_id = $1)
	`
	result, err := r.pool.Exec(ctx, deleteQuery, id, userID)
	if err != nil {
		return fmt.Errorf("deletar categoria: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Verifica se a categoria existia para diferenciar 404 de 409
		var exists bool
		err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tb_categories WHERE id = $1 AND user_id = $2)", id, userID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrCategoryNotFound
		}
		return ErrCategoryHasTransactions
	}

	return nil
}
