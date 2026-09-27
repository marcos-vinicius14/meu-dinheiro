package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/category"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	singletonPool   *pgxpool.Pool
	singletonRouter *chi.Mux
	singletonOnce   sync.Once
	internalAPIKey  = "test-internal-api-key"
	jwtSecret       = "test-jwt-secret-very-long-and-secure-2026"
)

type TestApp struct {
	Pool            *pgxpool.Pool
	Router          *chi.Mux
	UserRepo        *user.Repository
	BankAccountRepo *bankaccount.Repository
	CategoryRepo    *category.Repository
	TransactionRepo *transaction.Repository
	JWTService      *auth.JWTService
	AuthService     *auth.Service
}

func SetupTestApp(t *testing.T) *TestApp {
	ctx := context.Background()

	singletonOnce.Do(func() {
		// Inicia container singleton do PostgreSQL 18
		pgContainer, err := tcpostgres.Run(ctx,
			"postgres:18-alpine",
			tcpostgres.WithDatabase("meudinheiro_test"),
			tcpostgres.WithUsername("testuser"),
			tcpostgres.WithPassword("testpass"),
			tcpostgres.BasicWaitStrategies(),
		)
		require.NoError(t, err)

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err)

		// Executa migrações
		err = database.RunMigrations(connStr)
		require.NoError(t, err)

		// Conecta pool
		singletonPool, err = database.ConnectPool(ctx, connStr)
		require.NoError(t, err)
	})

	userRepo := user.NewRepository(singletonPool)
	bankAccountRepo := bankaccount.NewRepository(singletonPool)
	categoryRepo := category.NewRepository(singletonPool)
	transactionRepo := transaction.NewRepository(singletonPool)

	jwtService := auth.NewJWTService(jwtSecret)
	authService := auth.NewService(singletonPool, userRepo, jwtService, "MeuDinheiroTestBot")
	transactionService := transaction.NewService(transactionRepo, categoryRepo, bankAccountRepo)

	authHandler := auth.NewHandler(authService, internalAPIKey)
	bankAccountHandler := bankaccount.NewHandler(bankAccountRepo)
	categoryHandler := category.NewHandler(categoryRepo)
	transactionHandler := transaction.NewHandler(transactionService)

	r := web.NewRouter()
	authMiddleware := auth.RequireAuth(jwtService, userRepo, internalAPIKey)

	authHandler.RegisterRoutes(r, authMiddleware)
	bankAccountHandler.RegisterRoutes(r, authMiddleware)
	categoryHandler.RegisterRoutes(r, authMiddleware)
	transactionHandler.RegisterRoutes(r, authMiddleware)

	singletonRouter = r

	return &TestApp{
		Pool:            singletonPool,
		Router:          singletonRouter,
		UserRepo:        userRepo,
		BankAccountRepo: bankAccountRepo,
		CategoryRepo:    categoryRepo,
		TransactionRepo: transactionRepo,
		JWTService:      jwtService,
		AuthService:     authService,
	}
}

// CleanDatabase limpa dados de tabelas preservando a estrutura para isolamento entre testes.
func (app *TestApp) CleanDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		TRUNCATE TABLE tb_transactions, tb_transaction_bundles, tb_check_in_snapshots,
		               tb_bank_accounts, tb_categories, tb_telegram_auth_challenges, tb_users
		RESTART IDENTITY CASCADE
	`
	_, err := app.Pool.Exec(ctx, query)
	require.NoError(t, err)
}

// CreateTestUser cria um usuário de teste e retorna seus dados e token de sessão.
func (app *TestApp) CreateTestUser(t *testing.T, telegramID int64, firstName string) (*user.User, string) {
	ctx := context.Background()
	username := "test_user"
	u, err := app.UserRepo.FindOrCreateByTelegram(ctx, telegramID, &username, firstName)
	require.NoError(t, err)

	token, _, err := app.JWTService.GenerateToken(u)
	require.NoError(t, err)

	return u, token
}

// ExecuteRequest executa uma requisição HTTP contra o roteador in-memory.
func (app *TestApp) ExecuteRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	app.Router.ServeHTTP(rr, req)
	return rr
}

// FindFirstCategory busca a primeira categoria de um usuário (ex: Alimentação).
func (app *TestApp) FindFirstCategory(t *testing.T, userID uuid.UUID) uuid.UUID {
	categories, err := app.CategoryRepo.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	require.NotEmpty(t, categories)
	return categories[0].ID
}

// FindFirstBankAccount busca a primeira conta bancária de um usuário.
func (app *TestApp) FindFirstBankAccount(t *testing.T, userID uuid.UUID) uuid.UUID {
	accounts, err := app.BankAccountRepo.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	require.NotEmpty(t, accounts)
	return accounts[0].ID
}
