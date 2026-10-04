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

// APIClient é o cliente HTTP responsável pela comunicação entre o bot do Telegram e a API Meu Dinheiro.
type APIClient struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

// NewAPIClient instancia um novo APIClient com timeout configurado.
func NewAPIClient(baseURL, internalKey string) *APIClient {
	return &APIClient{
		baseURL:     baseURL,
		internalKey: internalKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AuthorizeChallenge autoriza o desafio de login Telegram emitido no frontend web.
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

// SaveOnboarding envia o payload inicial de configuração do usuário no encerramento do diálogo.
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

// GetUserContextByTelegram obtém a fotografia financeira consolidada do usuário (S2S, saldo líquido, ciclo, patrimônio).
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

// AddInvestment registra um novo aporte de investimento para o usuário.
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

// QuickExpense registra um gasto diário rápido com recalibração imediata do S2S.
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

// SimulatePurchase executa uma simulação what-if de compra (à vista ou até 48x) projetando o impacto em até 12 ciclos futuros.
func (c *APIClient) SimulatePurchase(ctx context.Context, req SimulationRequest) (*SimulationResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar simulacao: %w", err)
	}

	url := fmt.Sprintf("%s/internal/transactions/simulations", c.baseURL)
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

	var resp SimulationResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

// DailyCheckIn executa a conciliação diária de despesas não rastreadas e salva o snapshot diário no banco de forma idempotente.
func (c *APIClient) DailyCheckIn(ctx context.Context, req DailyCheckInRequest) (*DailyCheckInResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar checkin: %w", err)
	}

	url := fmt.Sprintf("%s/internal/transactions/checkin", c.baseURL)
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

	var resp DailyCheckInResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

// AdjustAccountBalance atualiza o saldo bancário da conta principal ou de uma conta específica durante o check-in.
func (c *APIClient) AdjustAccountBalance(ctx context.Context, req AdjustBalanceRequest) (*AdjustBalanceResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar ajuste de saldo: %w", err)
	}

	url := fmt.Sprintf("%s/internal/bank-accounts/balance", c.baseURL)
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

	var resp AdjustBalanceResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}

// RegisterIncome registra uma entrada de dinheiro (renda/receita) com recalibração imediata do S2S para cima.
func (c *APIClient) RegisterIncome(ctx context.Context, req IncomeRequest) (*IncomeResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("serializar receita: %w", err)
	}

	url := fmt.Sprintf("%s/internal/transactions/income", c.baseURL)
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

	var resp IncomeResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decodificar resposta: %w", err)
	}

	return &resp, nil
}
