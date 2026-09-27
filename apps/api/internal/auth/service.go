package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
)

var (
	ErrChallengeNotFound = errors.New("challenge não encontrado")
	ErrChallengeExpired  = errors.New("challenge expirado")
)

type ChallengeResult struct {
	Token     string    `json:"token"`
	BotURL    string    `json:"bot_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PollResult struct {
	Status    string     `json:"status"` // PENDING, AUTHORIZED, EXPIRED
	User      *user.User `json:"user,omitempty"`
	Token     string     `json:"token,omitempty"`
	ExpiresIn int64      `json:"expires_in,omitempty"`
}

type Service struct {
	pool        *pgxpool.Pool
	userRepo    *user.Repository
	jwtService  *JWTService
	botUsername string
}

func NewService(pool *pgxpool.Pool, userRepo *user.Repository, jwtService *JWTService, botUsername string) *Service {
	return &Service{
		pool:        pool,
		userRepo:    userRepo,
		jwtService:  jwtService,
		botUsername: botUsername,
	}
}

// CreateChallenge gera um novo token temporário de login para o Telegram.
func (s *Service) CreateChallenge(ctx context.Context) (*ChallengeResult, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("gerar token aleatório: %w", err)
	}
	token := hex.EncodeToString(bytes)
	expiresAt := time.Now().Add(5 * time.Minute)

	query := `
		INSERT INTO tb_telegram_auth_challenges (token, expires_at)
		VALUES ($1, $2)
	`
	_, err := s.pool.Exec(ctx, query, token, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("salvar challenge: %w", err)
	}

	botURL := fmt.Sprintf("https://t.me/%s?start=auth_%s", s.botUsername, token)

	return &ChallengeResult{
		Token:     token,
		BotURL:    botURL,
		ExpiresAt: expiresAt,
	}, nil
}

// AuthorizeChallenge é chamado pelo Bot do Telegram quando o usuário clica em /start auth_<token>.
func (s *Service) AuthorizeChallenge(
	ctx context.Context,
	token string,
	telegramID int64,
	username *string,
	firstName string,
) (*user.User, error) {
	// Verifica o challenge
	var challengeID uuid.UUID
	var status string
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT id, status, expires_at
		FROM tb_telegram_auth_challenges
		WHERE token = $1
	`, token).Scan(&challengeID, &status, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotFound
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		_, _ = s.pool.Exec(ctx, `UPDATE tb_telegram_auth_challenges SET status = 'EXPIRED' WHERE id = $1`, challengeID)
		return nil, ErrChallengeExpired
	}

	// Busca ou cria o usuário pelo telegram_id
	u, err := s.userRepo.FindOrCreateByTelegram(ctx, telegramID, username, firstName)
	if err != nil {
		return nil, fmt.Errorf("obter/criar usuário no challenge: %w", err)
	}

	// Atualiza o challenge para AUTHORIZED
	_, err = s.pool.Exec(ctx, `
		UPDATE tb_telegram_auth_challenges
		SET status = 'AUTHORIZED', user_id = $1
		WHERE id = $2
	`, u.ID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("atualizar challenge para autorizado: %w", err)
	}

	return u, nil
}

// PollChallenge é chamado pela Landing Page web para saber se o login já foi autorizado.
func (s *Service) PollChallenge(ctx context.Context, token string) (*PollResult, error) {
	var status string
	var expiresAt time.Time
	var userID *uuid.UUID

	err := s.pool.QueryRow(ctx, `
		SELECT status, expires_at, user_id
		FROM tb_telegram_auth_challenges
		WHERE token = $1
	`, token).Scan(&status, &expiresAt, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotFound
		}
		return nil, err
	}

	if status != "AUTHORIZED" && time.Now().After(expiresAt) {
		return &PollResult{Status: "EXPIRED"}, nil
	}

	if status != "AUTHORIZED" || userID == nil {
		return &PollResult{Status: "PENDING"}, nil
	}

	// Challenge autorizado: carrega o usuário e gera o token de sessão
	u, err := s.userRepo.FindByID(ctx, *userID)
	if err != nil {
		return nil, err
	}

	sessionToken, duration, err := s.jwtService.GenerateToken(u)
	if err != nil {
		return nil, err
	}

	return &PollResult{
		Status:    "AUTHORIZED",
		User:      u,
		Token:     sessionToken,
		ExpiresIn: int64(duration.Seconds()),
	}, nil
}

// CreateBotSession provisiona sessão para interação direta do Bot.
func (s *Service) CreateBotSession(
	ctx context.Context,
	telegramID int64,
	username *string,
	firstName string,
) (*user.User, string, error) {
	u, err := s.userRepo.FindOrCreateByTelegram(ctx, telegramID, username, firstName)
	if err != nil {
		return nil, "", err
	}

	token, _, err := s.jwtService.GenerateToken(u)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}
