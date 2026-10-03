package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

func NewAPIClient(baseURL, internalKey string) *APIClient {
	return &APIClient{
		baseURL:     baseURL,
		internalKey: internalKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type AuthorizeChallengeRequest struct {
	Token      string `json:"token"`
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
}

func (c *APIClient) AuthorizeChallenge(ctx context.Context, token string, telegramID int64, name string) error {
	reqBody := AuthorizeChallengeRequest{
		Token:      token,
		TelegramID: telegramID,
		Name:       name,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("serializar requisicao: %w", err)
	}

	url := fmt.Sprintf("%s/internal/auth/authorize-challenge", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("criar requisicao http: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	return nil
}

// --- TIPOS E MÉTODOS DO ONBOARDING, CONTEXTO E OPERAÇÃO DIÁRIA ---

type FixedExpenseInput struct {
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	CategoryName string  `json:"category_name"`
}

type InitialInvestmentInput struct {
	Ticker       string  `json:"ticker"`
	Quantity     float64 `json:"quantity"`
	AveragePrice float64 `json:"average_price"`
}

type OnboardingRequest struct {
	TelegramID          int64                    `json:"telegram_id"`
	FirstName           string                   `json:"first_name"`
	Username            *string                  `json:"username,omitempty"`
	InitialBalance      float64                  `json:"initial_balance"`
	CycleStartDay       int                      `json:"cycle_start_day"`
	FixedExpenses       []FixedExpenseInput      `json:"fixed_expenses"`
	EmergencyFundMonths int                      `json:"emergency_fund_months"`
	TargetSavings       float64                  `json:"target_savings"`
	FlexibleBudgetCap   float64                  `json:"flexible_budget_cap"`
	Investments         []InitialInvestmentInput `json:"investments"`
}

type CycleResponse struct {
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	DaysRemaining int     `json:"days_remaining"`
	S2SToday      float64 `json:"s2s_today"`
	HealthStatus  string  `json:"health_status"`
}

type EmergencyFundResponse struct {
	MonthlyEssentialCost float64 `json:"monthly_essential_cost"`
	Suggested6x          float64 `json:"suggested_6x"`
	Suggested12x         float64 `json:"suggested_12x"`
	ChosenTarget         float64 `json:"chosen_target"`
	ChosenMonths         int     `json:"chosen_months"`
	MonthsCovered        float64 `json:"months_covered"`
	ProgressPercent      float64 `json:"progress_percent"`
}

type OnboardingResponse struct {
	Message            string                `json:"message"`
	UserID             string                `json:"user_id"`
	Cycle              CycleResponse         `json:"cycle"`
	EmergencyFund      EmergencyFundResponse `json:"emergency_fund"`
	TotalLiquidBalance float64               `json:"total_liquid_balance"`
	TotalInvested      float64               `json:"total_invested"`
	TotalNetWorth      float64               `json:"total_net_worth"`
}

func (c *APIClient) SaveOnboarding(ctx context.Context, req OnboardingRequest) (*OnboardingResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar onboarding: %w", err)
	}

	url := fmt.Sprintf("%s/internal/users/onboarding", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var resp OnboardingResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

type UserFinancialContext struct {
	User struct {
		ID                  string  `json:"id"`
		TelegramID          int64   `json:"telegram_id"`
		FirstName           string  `json:"first_name"`
		Username            *string `json:"username,omitempty"`
		TargetSavings       float64 `json:"target_savings"`
		FlexibleBudgetCap   float64 `json:"flexible_budget_cap"`
		EmergencyFundTarget float64 `json:"emergency_fund_target"`
		EmergencyFundMonths int     `json:"emergency_fund_months"`
		CycleStartDay       int     `json:"cycle_start_day"`
	} `json:"user"`
	Accounts []struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		CurrentBalance float64 `json:"current_balance"`
	} `json:"accounts"`
	TotalLiquidBalance float64 `json:"total_liquid_balance"`
	Cycle              struct {
		StartDate         string  `json:"start_date"`
		EndDate           string  `json:"end_date"`
		DaysRemaining     int     `json:"days_remaining"`
		S2SToday          float64 `json:"s2s_today"`
		HealthStatus      string  `json:"health_status"`
		ProjectedBalance  float64 `json:"projected_balance"`
		RemainingFlexible float64 `json:"remaining_flexible"`
		FlexibleSpent     float64 `json:"flexible_spent"`
	} `json:"cycle"`
	EmergencyFund struct {
		MonthlyEssentialCost float64 `json:"monthly_essential_cost"`
		Target               float64 `json:"target"`
		Months               int     `json:"months"`
		MonthsCovered        float64 `json:"months_covered"`
		ProgressPercent      float64 `json:"progress_percent"`
	} `json:"emergency_fund"`
	Investments []struct {
		ID           string  `json:"id"`
		Ticker       string  `json:"ticker"`
		Quantity     float64 `json:"quantity"`
		AveragePrice float64 `json:"average_price"`
		TotalCost    float64 `json:"total_cost"`
	} `json:"investments"`
	TotalInvested float64 `json:"total_invested"`
	TotalNetWorth float64 `json:"total_net_worth"`
}

func (c *APIClient) GetUserContextByTelegram(ctx context.Context, telegramID int64) (*UserFinancialContext, error) {
	url := fmt.Sprintf("%s/internal/users/context-by-telegram?telegram_id=%d", c.baseURL, telegramID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var ctxResp UserFinancialContext
	if err := json.NewDecoder(res.Body).Decode(&ctxResp); err != nil {
		return nil, fmt.Errorf("decodificar contexto: %w", err)
	}

	return &ctxResp, nil
}

type AddInvestmentRequest struct {
	TelegramID int64   `json:"telegram_id"`
	Ticker     string  `json:"ticker"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
}

type AddInvestmentResponse struct {
	Investment struct {
		ID           string  `json:"id"`
		Ticker       string  `json:"ticker"`
		Quantity     float64 `json:"quantity"`
		AveragePrice float64 `json:"average_price"`
		TotalCost    float64 `json:"total_cost"`
	} `json:"investment"`
	TotalInvested float64 `json:"total_invested"`
	TotalNetWorth float64 `json:"total_net_worth"`
}

func (c *APIClient) AddInvestment(ctx context.Context, req AddInvestmentRequest) (*AddInvestmentResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar ativo: %w", err)
	}

	url := fmt.Sprintf("%s/internal/investments", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var resp AddInvestmentResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

type QuickExpenseRequest struct {
	TelegramID   int64   `json:"telegram_id"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	CategoryName string  `json:"category_name,omitempty"`
	Date         string  `json:"date,omitempty"`
}

type QuickExpenseResponse struct {
	Transaction struct {
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	} `json:"transaction"`
	CategoryName  string  `json:"category_name"`
	PreviousS2S   float64 `json:"previous_s2s"`
	NewS2S        float64 `json:"new_s2s"`
	HealthStatus  string  `json:"health_status"`
	DaysRemaining int     `json:"days_remaining"`
}

func (c *APIClient) QuickExpense(ctx context.Context, req QuickExpenseRequest) (*QuickExpenseResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar despesa: %w", err)
	}

	url := fmt.Sprintf("%s/internal/transactions/quick-expense", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("criar requisicao http: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.internalKey)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar com api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("erro da api (status %d): %s", res.StatusCode, string(body))
	}

	var resp QuickExpenseResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}
