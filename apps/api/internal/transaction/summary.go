package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/shopspring/decimal"
)

type DailyExpenseItem struct {
	ID           uuid.UUID              `json:"id"`
	Description  string                 `json:"description"`
	Amount       money.Money            `json:"amount"`
	CategoryName string                 `json:"category_name"`
	Type         engine.TransactionType `json:"type"`
	Date         time.Time              `json:"date"`
}

type DailyExpensesSummary struct {
	Date         time.Time          `json:"date"`
	TotalSpent   money.Money        `json:"total_spent"`
	Count        int                `json:"count"`
	Transactions []DailyExpenseItem `json:"transactions"`
}

type CategoryExpenseSummary struct {
	CategoryID   uuid.UUID       `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Amount       money.Money     `json:"amount"`
	Count        int             `json:"count"`
	Percentage   decimal.Decimal `json:"percentage"`
}

type WeeklyExpensesSummary struct {
	StartDate     time.Time                `json:"start_date"`
	EndDate       time.Time                `json:"end_date"`
	DaysElapsed   int                      `json:"days_elapsed"`
	TotalSpent    money.Money              `json:"total_spent"`
	DailyAverage  money.Money              `json:"daily_average"`
	Count         int                      `json:"count"`
	TopCategories []CategoryExpenseSummary `json:"top_categories"`
}

// GetDailyExpensesSummary recupera a lista e agregação de despesas confirmadas do dia especificado.
func (r *Repository) GetDailyExpensesSummary(ctx context.Context, userID uuid.UUID, targetDate time.Time) (*DailyExpensesSummary, error) {
	normDate := dateinterval.NormalizeDate(targetDate)
	query := `
		SELECT t.id, t.description, t.value, t.type, c.description AS category_name,
		       COALESCE(t.payment_date, t.due_date) AS tx_date
		FROM tb_transactions t
		JOIN tb_categories c ON t.category_id = c.id
		WHERE t.user_id = $1
		  AND t.type IN ('FIXED_EXPENSE', 'FLEXIBLE_EXPENSE', 'INSTALLMENT_EXPENSE')
		  AND t.status = 'CONFIRMED'
		  AND (t.payment_date = $2 OR (t.payment_date IS NULL AND t.due_date = $2))
		ORDER BY t.id DESC
	`
	rows, err := r.pool.Query(ctx, query, userID, normDate)
	if err != nil {
		return nil, fmt.Errorf("consultar extrato diário: %w", err)
	}
	defer rows.Close()

	totalSpent := money.Zero()
	items := make([]DailyExpenseItem, 0)
	for rows.Next() {
		var item DailyExpenseItem
		if err := rows.Scan(
			&item.ID,
			&item.Description,
			&item.Amount,
			&item.Type,
			&item.CategoryName,
			&item.Date,
		); err != nil {
			return nil, fmt.Errorf("ler item do extrato diário: %w", err)
		}
		totalSpent = totalSpent.Add(item.Amount)
		items = append(items, item)
	}

	return &DailyExpensesSummary{
		Date:         normDate,
		TotalSpent:   totalSpent,
		Count:        len(items),
		Transactions: items,
	}, nil
}

// GetWeeklyExpensesSummary calcula a agregação semanal consolidada e ranking de top categorias.
func (r *Repository) GetWeeklyExpensesSummary(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time, daysElapsed int) (*WeeklyExpensesSummary, error) {
	normStart := dateinterval.NormalizeDate(startDate)
	normEnd := dateinterval.NormalizeDate(endDate)
	if daysElapsed < 1 {
		daysElapsed = 1
	}

	query := `
		SELECT c.id, c.description AS category_name, COALESCE(SUM(t.value), 0) AS total_amount, COUNT(t.id) AS tx_count
		FROM tb_transactions t
		JOIN tb_categories c ON t.category_id = c.id
		WHERE t.user_id = $1
		  AND t.type IN ('FIXED_EXPENSE', 'FLEXIBLE_EXPENSE', 'INSTALLMENT_EXPENSE')
		  AND t.status = 'CONFIRMED'
		  AND COALESCE(t.payment_date, t.due_date) >= $2
		  AND COALESCE(t.payment_date, t.due_date) <= $3
		GROUP BY c.id, c.description
		ORDER BY total_amount DESC, c.description ASC
	`
	rows, err := r.pool.Query(ctx, query, userID, normStart, normEnd)
	if err != nil {
		return nil, fmt.Errorf("consultar resumo semanal por categoria: %w", err)
	}
	defer rows.Close()

	totalSpent := money.Zero()
	totalCount := 0
	categories := make([]CategoryExpenseSummary, 0)

	for rows.Next() {
		var cat CategoryExpenseSummary
		if err := rows.Scan(
			&cat.CategoryID,
			&cat.CategoryName,
			&cat.Amount,
			&cat.Count,
		); err != nil {
			return nil, fmt.Errorf("ler categoria do resumo semanal: %w", err)
		}
		totalSpent = totalSpent.Add(cat.Amount)
		totalCount += cat.Count
		categories = append(categories, cat)
	}

	// Calcula percentual de cada categoria
	for i := range categories {
		if totalSpent.IsPositive() {
			categories[i].Percentage = categories[i].Amount.PercentOver(totalSpent)
		}
	}

	dailyAverage, _ := totalSpent.Divide(daysElapsed)

	topCategories := categories
	if len(topCategories) > 5 {
		topCategories = topCategories[:5]
	}

	return &WeeklyExpensesSummary{
		StartDate:     normStart,
		EndDate:       normEnd,
		DaysElapsed:   daysElapsed,
		TotalSpent:    totalSpent,
		DailyAverage:  dailyAverage,
		Count:         totalCount,
		TopCategories: topCategories,
	}, nil
}
