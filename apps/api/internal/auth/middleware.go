package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type contextKey string

const userCtxKey contextKey = "authenticated_user"

func RequireAuth(jwtService *JWTService, userRepo *user.Repository, internalAPIKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Verifica se é chamada interna autorizada do Bot com X-Telegram-User-ID
			internalKey := r.Header.Get("X-Internal-Secret")
			if internalKey == "" {
				internalKey = r.Header.Get("X-Internal-API-Key")
			}
			if internalKey != "" && internalKey == internalAPIKey {
				if tgIDStr := r.Header.Get("X-Telegram-User-ID"); tgIDStr != "" {
					tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
					if err == nil {
						u, err := userRepo.FindByTelegramID(r.Context(), tgID)
						if err == nil && u != nil {
							ctx := context.WithValue(r.Context(), userCtxKey, u)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			// 2. Extrai token do cookie "session_token"
			var tokenString string
			if cookie, err := r.Cookie("session_token"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			}

			if tokenString == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if tokenString == "" {
				web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
				return
			}

			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
				return
			}

			u, err := userRepo.FindByID(r.Context(), claims.UserID)
			if err != nil {
				web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(userCtxKey).(*user.User)
	return u, ok
}
