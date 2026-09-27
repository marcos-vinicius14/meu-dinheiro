package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
)

var (
	ErrInvalidToken = errors.New("token de autenticação inválido ou expirado")
	sessionDuration = 30 * 24 * time.Hour
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

type UserClaims struct {
	UserID     uuid.UUID `json:"sub"`
	TelegramID int64     `json:"telegram_id"`
	jwt.RegisteredClaims
}

func (s *JWTService) GenerateToken(u *user.User) (string, time.Duration, error) {
	now := time.Now()
	claims := UserClaims{
		UserID:     u.ID,
		TelegramID: u.TelegramID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionDuration)),
			Issuer:    "meudinheiro",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("assinar token JWT: %w", err)
	}

	return tokenString, sessionDuration, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
