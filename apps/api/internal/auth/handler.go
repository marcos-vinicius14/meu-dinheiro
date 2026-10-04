package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type Handler struct {
	service        *Service
	internalAPIKey string
}

func NewHandler(service *Service, internalAPIKey string) *Handler {
	return &Handler{
		service:        service,
		internalAPIKey: internalAPIKey,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Rotas públicas de autenticação via Telegram
	r.Post("/auth/telegram/challenge", h.handleCreateChallenge)
	r.Get("/auth/telegram/poll", h.handlePollChallenge)
	r.Post("/auth/logout", h.handleLogout)

	// Rotas autenticadas do usuário
	r.Group(func(protected chi.Router) {
		protected.Use(authMiddleware)
		protected.Get("/auth/me", h.handleMe)
	})

	// Rotas internas exclusivas do Bot do Telegram
	r.Group(func(internal chi.Router) {
		internal.Use(h.requireInternalKey)
		internal.Post("/internal/auth/authorize-challenge", h.handleAuthorizeChallenge)
		internal.Post("/internal/auth/bot-session", h.handleBotSession)
	})
}

func (h *Handler) handleCreateChallenge(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.CreateChallenge(r.Context())
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao criar challenge de autenticação")
		return
	}
	web.JSON(w, http.StatusCreated, result)
}

func (h *Handler) handlePollChallenge(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		web.Error(w, http.StatusBadRequest, "Token de challenge é obrigatório")
		return
	}

	result, err := h.service.PollChallenge(r.Context(), token)
	if err != nil {
		if err == ErrChallengeNotFound {
			web.Error(w, http.StatusNotFound, "Challenge não encontrado")
			return
		}
		web.Error(w, http.StatusInternalServerError, "Erro ao consultar challenge")
		return
	}

	// Se o challenge foi autorizado, define o cookie de sessão na resposta
	if result.Status == "AUTHORIZED" && result.Token != "" {
		web.SetSessionCookie(w, result.Token, time.Duration(result.ExpiresIn)*time.Second)
	}

	web.JSON(w, http.StatusOK, result)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}
	web.JSON(w, http.StatusOK, u)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	web.ClearSessionCookie(w)
	web.NoContent(w)
}

type authorizeChallengeRequest struct {
	Token      string  `json:"token"`
	TelegramID int64   `json:"telegram_id"`
	Username   *string `json:"username"`
	FirstName  string  `json:"first_name"`
}

func (h *Handler) handleAuthorizeChallenge(w http.ResponseWriter, r *http.Request) {
	var req authorizeChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.Token == "" || req.TelegramID == 0 || req.FirstName == "" {
		web.Error(w, http.StatusBadRequest, "token, telegram_id e first_name são obrigatórios")
		return
	}

	u, err := h.service.AuthorizeChallenge(r.Context(), req.Token, req.TelegramID, req.Username, req.FirstName)
	if err != nil {
		if err == ErrChallengeNotFound {
			web.Error(w, http.StatusNotFound, "Challenge não encontrado")
			return
		}
		if err == ErrChallengeExpired {
			web.Error(w, http.StatusBadRequest, "Challenge expirado")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, map[string]any{
		"message": "Challenge autorizado com sucesso",
		"user":    u,
	})
}

type botSessionRequest struct {
	TelegramID int64   `json:"telegram_id"`
	Username   *string `json:"username"`
	FirstName  string  `json:"first_name"`
}

func (h *Handler) handleBotSession(w http.ResponseWriter, r *http.Request) {
	var req botSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}
	if req.TelegramID == 0 || req.FirstName == "" {
		web.Error(w, http.StatusBadRequest, "telegram_id e first_name são obrigatórios")
		return
	}

	u, token, err := h.service.CreateBotSession(r.Context(), req.TelegramID, req.Username, req.FirstName)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao criar sessão para o bot")
		return
	}

	web.JSON(w, http.StatusOK, map[string]any{
		"user":  u,
		"token": token,
	})
}

func (h *Handler) requireInternalKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Internal-Secret")
		if key == "" {
			key = r.Header.Get("X-Internal-API-Key")
		}
		if key == "" || key != h.internalAPIKey {
			web.Error(w, http.StatusForbidden, "Acesso restrito: chave interna inválida")
			return
		}
		next.ServeHTTP(w, r)
	})
}
