package category

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(protected chi.Router) {
		protected.Use(authMiddleware)
		protected.Post("/categories", h.handleCreate)
		protected.Get("/categories", h.handleList)
		protected.Get("/categories/{id}", h.handleFind)
		protected.Put("/categories/{id}", h.handleUpdate)
		protected.Delete("/categories/{id}", h.handleDelete)
	})
}

type createRequest struct {
	Description string  `json:"description"`
	Icon        *string `json:"icon"`
	IsFlexible  bool    `json:"is_flexible"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	c, err := h.repo.Create(r.Context(), u.ID, req.Description, req.Icon, req.IsFlexible)
	if err != nil {
		switch err {
		case ErrCategoryDuplicated, ErrInvalidDescription:
			web.Error(w, http.StatusBadRequest, err.Error())
		default:
			web.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	web.JSON(w, http.StatusCreated, c)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	categories, err := h.repo.ListByUserID(r.Context(), u.ID)
	if err != nil {
		web.Error(w, http.StatusInternalServerError, "Erro ao listar categorias")
		return
	}

	web.JSON(w, http.StatusOK, categories)
}

func (h *Handler) handleFind(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de categoria inválido")
		return
	}

	c, err := h.repo.FindByIDAndUserID(r.Context(), id, u.ID)
	if err != nil {
		if err == ErrCategoryNotFound {
			web.Error(w, http.StatusNotFound, "Categoria não encontrada")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, c)
}

type updateRequest struct {
	Description string  `json:"description"`
	Icon        *string `json:"icon"`
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de categoria inválido")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	c, err := h.repo.Update(r.Context(), id, u.ID, req.Description, req.Icon)
	if err != nil {
		if err == ErrCategoryNotFound {
			web.Error(w, http.StatusNotFound, "Categoria não encontrada")
			return
		}
		if err == ErrCategoryDuplicated || err == ErrInvalidDescription {
			web.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.JSON(w, http.StatusOK, c)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u == nil {
		web.Error(w, http.StatusUnauthorized, "Autenticação necessária")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, http.StatusBadRequest, "ID de categoria inválido")
		return
	}

	err = h.repo.Delete(r.Context(), id, u.ID)
	if err != nil {
		if err == ErrCategoryNotFound {
			web.Error(w, http.StatusNotFound, "Categoria não encontrada")
			return
		}
		if err == ErrCategoryHasTransactions {
			web.Error(w, http.StatusConflict, "Categoria possui transações vinculadas")
			return
		}
		web.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	web.NoContent(w)
}
