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
	baseURL    string
	internalKey string
	httpClient *http.Client
}

func NewAPIClient(baseURL, internalKey string) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
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
