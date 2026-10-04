package botapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/category"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/investment"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction/engine"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

type Handler struct {
	internalKey     string
	userRepo        *user.Repository
	bankAccountRepo *bankaccount.Repository
	categoryRepo    *category.Repository
	transactionRepo *transaction.Repository
	investService   *investment.Service
	txService       *transaction.Service
	predictiveEng   *engine.PredictiveEngine
}

func NewHandler(
	internalKey string,
	userRepo *user.Repository,
	bankAccountRepo *bankaccount.Repository,
	categoryRepo *category.Repository,
	transactionRepo *transaction.Repository,
	investService *investment.Service,
	txService *transaction.Service,
) *Handler {
	return &Handler{
		internalKey:     internalKey,
		userRepo:        userRepo,
		bankAccountRepo: bankAccountRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
		investService:   investService,
		txService:       txService,
		predictiveEng:   engine.NewPredictiveEngine(),
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Group(func(internal chi.Router) {
		internal.Use(h.requireInternalSecret)
		internal.Post("/internal/users/onboarding", h.handleOnboarding)
		internal.Get("/internal/users/context-by-telegram", h.handleContextByTelegram)
		internal.Post("/internal/investments", h.handleAddInvestment)
		internal.Post("/internal/transactions/quick-expense", h.handleQuickExpense)
		internal.Post("/internal/transactions/simulations", h.handleSimulatePurchase)
		internal.Post("/internal/transactions/checkin", h.handleDailyCheckIn)
		internal.Post("/internal/transactions/income", h.handleIncome)
		internal.Post("/internal/bank-accounts/balance", h.handleAdjustBalance)
	})
}

// requireInternalSecret aceita tanto X-Internal-Secret quanto X-Internal-API-Key para interoperabilidade total.
func (h *Handler) requireInternalSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get("X-Internal-Secret")
		if secret == "" {
			secret = r.Header.Get("X-Internal-API-Key")
		}
		if secret == "" || secret != h.internalKey {
			web.Error(w, http.StatusForbidden, "Acesso restrito: chave interna inválida")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// matchCategory busca categoria por nome case-insensitive ou retorna a primeira flexível (ou primeira geral).
func matchCategory(categories []category.Category, targetName string) uuid.UUID {
	if len(categories) == 0 {
		return uuid.Nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(targetName))
	if trimmed != "" {
		for _, c := range categories {
			if strings.ToLower(c.Description) == trimmed {
				return c.ID
			}
		}
	}

	// Fallback para primeira categoria flexível (ex: Alimentação)
	for _, c := range categories {
		if c.IsFlexible {
			return c.ID
		}
	}

	return categories[0].ID
}
