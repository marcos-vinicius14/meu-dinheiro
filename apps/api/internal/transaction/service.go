package transaction

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/category"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/dateinterval"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/shopspring/decimal"
)

type Service struct {
	repo            *Repository
	categoryRepo    *category.Repository
	bankAccountRepo *bankaccount.Repository
	predictiveEng   *engine.PredictiveEngine
	simulator       *engine.WhatIfSimulator
}

func NewService(
	repo *Repository,
	categoryRepo *category.Repository,
	bankAccountRepo *bankaccount.Repository,
) *Service {
	eng := engine.NewPredictiveEngine()
	return &Service{
		repo:            repo,
		categoryRepo:    categoryRepo,
		bankAccountRepo: bankAccountRepo,
		predictiveEng:   eng,
		simulator:       engine.NewWhatIfSimulator(eng),
	}
}

func (s *Service) CreateTransaction(
	ctx context.Context,
	userID uuid.UUID,
	description string,
	amount money.Money,
	txType engine.TransactionType,
	dueDate time.Time,
	categoryID uuid.UUID,
	bankAccountID *uuid.UUID,
) (*Transaction, error) {
	if txType == engine.TypeInstallmentExpense {
		return nil, ErrInstallmentTypeResticted
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, ErrAmountMustBePositive
	}
	if dueDate.IsZero() {
		return nil, ErrDueDateRequired
	}

	// Valida categoria pertencente ao usuário
	if _, err := s.categoryRepo.FindByIDAndUserID(ctx, categoryID, userID); err != nil {
		return nil, err
	}

	// Valida conta bancária se informada
	if bankAccountID != nil {
		if _, err := s.bankAccountRepo.FindByIDAndUserID(ctx, *bankAccountID, userID); err != nil {
			return nil, err
		}
	}

	tx := &Transaction{
		UserID:        userID,
		BankAccountID: bankAccountID,
		CategoryID:    categoryID,
		Description:   description,
		Amount:        amount,
		Type:          txType,
		Status:        engine.StatusProjected,
		DueDate:       dueDate,
	}

	return s.repo.Create(ctx, tx)
}

func (s *Service) ListTransactions(ctx context.Context, userID uuid.UUID) ([]Transaction, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) FindTransaction(ctx context.Context, id, userID uuid.UUID) (*Transaction, error) {
	return s.repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *Service) UpdateTransaction(
	ctx context.Context,
	id, userID uuid.UUID,
	description string,
	amount money.Money,
	txType engine.TransactionType,
	dueDate time.Time,
	categoryID uuid.UUID,
	bankAccountID *uuid.UUID,
) (*Transaction, error) {
	existing, err := s.repo.FindByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.BundleID != nil {
		return nil, ErrInstallmentImmutable
	}
	if txType == engine.TypeInstallmentExpense {
		return nil, ErrInstallmentTypeResticted
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, ErrAmountMustBePositive
	}
	if dueDate.IsZero() {
		return nil, ErrDueDateRequired
	}

	if _, err := s.categoryRepo.FindByIDAndUserID(ctx, categoryID, userID); err != nil {
		return nil, err
	}

	if bankAccountID != nil {
		if _, err := s.bankAccountRepo.FindByIDAndUserID(ctx, *bankAccountID, userID); err != nil {
			return nil, err
		}
	}

	existing.Description = description
	existing.Amount = amount
	existing.Type = txType
	existing.DueDate = dueDate
	existing.CategoryID = categoryID
	existing.BankAccountID = bankAccountID

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *Service) DeleteTransaction(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *Service) CreateBundle(
	ctx context.Context,
	userID uuid.UUID,
	description string,
	totalAmount money.Money,
	totalInstallments int,
	firstDueDate time.Time,
	categoryID uuid.UUID,
	bankAccountID *uuid.UUID,
) (*Bundle, []Transaction, error) {
	if totalInstallments <= 0 {
		return nil, nil, ErrInstallmentsRequired
	}
	if totalAmount.IsNegative() || totalAmount.IsZero() {
		return nil, nil, ErrAmountMustBePositive
	}
	if firstDueDate.IsZero() {
		return nil, nil, ErrFirstDueDateRequired
	}

	if _, err := s.categoryRepo.FindByIDAndUserID(ctx, categoryID, userID); err != nil {
		return nil, nil, err
	}
	if bankAccountID != nil {
		if _, err := s.bankAccountRepo.FindByIDAndUserID(ctx, *bankAccountID, userID); err != nil {
			return nil, nil, err
		}
	}

	allocations, err := totalAmount.Allocate(totalInstallments)
	if err != nil {
		return nil, nil, err
	}

	bundle := &Bundle{
		UserID:            userID,
		Description:       description,
		TotalAmount:       totalAmount,
		TotalInstallments: totalInstallments,
		FirstDueDate:      firstDueDate,
	}

	installments := make([]Transaction, totalInstallments)
	normFirstDate := dateinterval.NormalizeDate(firstDueDate)

	for i := 0; i < totalInstallments; i++ {
		instNumber := i + 1
		// Vencimentos mensais sucessivos preservando day-of-month clampado
		instDueDate := addMonthsClamped(normFirstDate, i)

		installments[i] = Transaction{
			UserID:            userID,
			BankAccountID:     bankAccountID,
			CategoryID:        categoryID,
			Description:       description,
			Amount:            allocations[i],
			Type:              engine.TypeInstallmentExpense,
			Status:            engine.StatusCommitted,
			DueDate:           instDueDate,
			InstallmentNumber: &instNumber,
			TotalInstallments: &totalInstallments,
		}
	}

	return s.repo.CreateBundleWithInstallments(ctx, bundle, installments)
}

type UntrackedExpenseInput struct {
	Description string      `json:"description"`
	Amount      money.Money `json:"amount"`
	CategoryID  uuid.UUID   `json:"category_id"`
}

type DailyCheckInInput struct {
	Date                           time.Time               `json:"date"`
	LiquidBalance                  money.Money             `json:"liquid_balance"`
	TargetSavings                  money.Money             `json:"target_savings"`
	FlexibleBudgetCap              money.Money             `json:"flexible_budget_cap"`
	UntrackedExpenses              []UntrackedExpenseInput `json:"untracked_expenses"`
	ConfirmedPendingTransactionIDs []uuid.UUID             `json:"confirmed_pending_transaction_ids"`
}

type DailyCheckInResult struct {
	S2SCalculated        money.Money         `json:"s2s_calculated"`
	SpentToday           money.Money         `json:"spent_today"`
	DeltaFromSafeToSpend money.Money         `json:"delta_from_safe_to_spend"`
	HealthStatus         engine.HealthStatus `json:"health_status"`
	NextDayS2S           money.Money         `json:"next_day_s2s"`
	ProjectedFreeBalance money.Money         `json:"projected_free_balance"`
	DaysRemaining        int                 `json:"days_remaining"`
}

func (s *Service) DailyCheckIn(ctx context.Context, userID uuid.UUID, input DailyCheckInInput) (*DailyCheckInResult, error) {
	if input.Date.IsZero() {
		return nil, ErrCheckInDateRequired
	}

	// 1. Pré-validação atômica de categorias
	newExpenses := make([]Transaction, len(input.UntrackedExpenses))
	spentToday := money.Zero()

	for i, exp := range input.UntrackedExpenses {
		if _, err := s.categoryRepo.FindByIDAndUserID(ctx, exp.CategoryID, userID); err != nil {
			return nil, err
		}
		if exp.Amount.IsNegative() || exp.Amount.IsZero() {
			return nil, ErrAmountMustBePositive
		}
		newExpenses[i] = Transaction{
			UserID:      userID,
			CategoryID:  exp.CategoryID,
			Description: exp.Description,
			Amount:      exp.Amount,
		}
		spentToday = spentToday.Add(exp.Amount)
	}

	// 2. Pré-validação de pendentes
	for _, txID := range input.ConfirmedPendingTransactionIDs {
		tx, err := s.repo.FindByIDAndUserID(ctx, txID, userID)
		if err != nil {
			return nil, err
		}
		if tx.Status == engine.StatusCanceled {
			return nil, ErrCanceledCannotConfirm
		}
	}

	// 3. Persistência atômica
	if err := s.repo.ExecuteCheckInWrites(ctx, userID, newExpenses, input.ConfirmedPendingTransactionIDs, input.Date); err != nil {
		return nil, err
	}

	// 4. Carrega snapshots e roda a engine para hoje
	snapshots, err := s.repo.LoadSnapshotsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	contextToday := engine.CycleContext{
		CycleInterval:     dateinterval.MonthOf(input.Date),
		CurrentDate:       input.Date,
		TargetSavings:     input.TargetSavings,
		FlexibleBudgetCap: input.FlexibleBudgetCap,
	}

	stateToday := engine.CurrentState{
		LiquidBalance: input.LiquidBalance,
		Transactions:  snapshots,
	}

	engineResult := s.predictiveEng.Calculate(contextToday, stateToday)
	delta := engineResult.S2SToday.Subtract(spentToday)

	// 5. Upsert do snapshot no banco
	snapshotModel := &CheckInSnapshot{
		UserID:               userID,
		CheckInDate:          input.Date,
		S2SCalculated:        engineResult.S2SToday,
		SpentToday:           spentToday,
		DeltaFromSafeToSpend: delta,
		HealthStatus:         engineResult.HealthStatus,
	}
	if err := s.repo.UpsertSnapshot(ctx, snapshotModel); err != nil {
		return nil, err
	}

	// 6. Recalibra S2S do dia seguinte (D+1)
	tomorrow := dateinterval.NormalizeDate(input.Date).AddDate(0, 0, 1)
	contextTomorrow := engine.CycleContext{
		CycleInterval:     dateinterval.MonthOf(input.Date),
		CurrentDate:       tomorrow,
		TargetSavings:     input.TargetSavings,
		FlexibleBudgetCap: input.FlexibleBudgetCap,
	}
	nextDayResult := s.predictiveEng.Calculate(contextTomorrow, stateToday)

	// 7. Projected Free Balance
	// projectedFreeBalance = netLiquidityBeforeFlex - max(0, flexibleBudgetCap - flexSpent)
	remainingBudget := money.Max(input.FlexibleBudgetCap.Subtract(engineResult.FlexSpent), money.Zero())
	projectedFreeBalance := engineResult.NetLiquidityBeforeFlex.Subtract(remainingBudget)

	return &DailyCheckInResult{
		S2SCalculated:        engineResult.S2SToday,
		SpentToday:           spentToday,
		DeltaFromSafeToSpend: delta,
		HealthStatus:         engineResult.HealthStatus,
		NextDayS2S:           nextDayResult.S2SToday,
		ProjectedFreeBalance: projectedFreeBalance,
		DaysRemaining:        engineResult.DaysRemaining,
	}, nil
}

type SimulationInput struct {
	Date              time.Time   `json:"date"`
	LiquidBalance     money.Money `json:"liquid_balance"`
	TargetSavings     money.Money `json:"target_savings"`
	FlexibleBudgetCap money.Money `json:"flexible_budget_cap"`
	TotalAmount       money.Money `json:"total_amount"`
	Installments      int         `json:"installments"`
	FirstDueDate      time.Time   `json:"first_due_date"`
}

type SimulationCycleDto struct {
	CycleStart          time.Time           `json:"cycle_start"`
	CycleEnd            time.Time           `json:"cycle_end"`
	S2SToday            money.Money         `json:"s2s_today"`
	S2SReduction        money.Money         `json:"s2s_reduction"`
	S2SReductionPercent decimal.Decimal     `json:"s2s_reduction_percent"`
	ProjectedBalance    money.Money         `json:"projected_balance"`
	HealthStatus        engine.HealthStatus `json:"health_status"`
	Bottleneck          bool                `json:"bottleneck"`
}

func (s *Service) SimulatePurchase(ctx context.Context, userID uuid.UUID, input SimulationInput) ([]SimulationCycleDto, error) {
	if input.Date.IsZero() {
		return nil, ErrSimulationDateRequired
	}
	if input.Installments <= 0 {
		return nil, ErrInstallmentsRequired
	}
	if input.FirstDueDate.IsZero() {
		return nil, ErrFirstDueDateRequired
	}

	cycleInterval := dateinterval.MonthOf(input.Date)
	context := engine.CycleContext{
		CycleInterval:     cycleInterval,
		CurrentDate:       input.Date,
		TargetSavings:     input.TargetSavings,
		FlexibleBudgetCap: input.FlexibleBudgetCap,
	}

	state := engine.CurrentState{
		LiquidBalance: input.LiquidBalance,
		Transactions:  nil,
	}

	purchase := engine.SimulatedPurchase{
		TotalAmount:  input.TotalAmount,
		Installments: input.Installments,
		FirstDueDate: input.FirstDueDate,
	}

	simulations, err := s.simulator.Simulate(context, state, purchase)
	if err != nil {
		return nil, err
	}

	dtos := make([]SimulationCycleDto, len(simulations))
	for i, sim := range simulations {
		dtos[i] = SimulationCycleDto{
			CycleStart:          sim.CycleInterval.StartDate(),
			CycleEnd:            sim.CycleInterval.EndDate(),
			S2SToday:            sim.EngineResult.S2SToday,
			S2SReduction:        sim.S2SReduction,
			S2SReductionPercent: sim.S2SReductionPercent,
			ProjectedBalance:    sim.EngineResult.ProjectedBalance,
			HealthStatus:        sim.HealthStatus,
			Bottleneck:          sim.Bottleneck,
		}
	}

	return dtos, nil
}

func addMonthsClamped(date time.Time, months int) time.Time {
	targetYear := date.Year()
	targetMonth := date.Month() + time.Month(months)
	targetDay := date.Day()

	// Normaliza mês/ano se ultrapassar 12
	firstOfTargetMonth := time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, time.UTC)
	lastDayOfTargetMonth := time.Date(firstOfTargetMonth.Year(), firstOfTargetMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

	if targetDay > lastDayOfTargetMonth {
		targetDay = lastDayOfTargetMonth
	}

	return time.Date(firstOfTargetMonth.Year(), firstOfTargetMonth.Month(), targetDay, 0, 0, 0, 0, time.UTC)
}
