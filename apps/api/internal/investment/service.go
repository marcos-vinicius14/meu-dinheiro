package investment

import (
	"context"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/money"
	"github.com/shopspring/decimal"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type PortfolioResponse struct {
	Investments   []Investment `json:"investments"`
	TotalInvested money.Money  `json:"total_invested"`
}

func (s *Service) AddLot(
	ctx context.Context,
	userID uuid.UUID,
	ticker string,
	quantity decimal.Decimal,
	price money.Money,
) (*Investment, error) {
	return s.repo.AddLot(ctx, userID, ticker, quantity, price)
}

func (s *Service) Sell(
	ctx context.Context,
	userID uuid.UUID,
	ticker string,
	quantity decimal.Decimal,
) (*Investment, error) {
	return s.repo.Sell(ctx, userID, ticker, quantity)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) (*PortfolioResponse, error) {
	list, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CalculateTotalInvested(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &PortfolioResponse{
		Investments:   list,
		TotalInvested: total,
	}, nil
}

func (s *Service) Find(ctx context.Context, id, userID uuid.UUID) (*Investment, error) {
	return s.repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}
