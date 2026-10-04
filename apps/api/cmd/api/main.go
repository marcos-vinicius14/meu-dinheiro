package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/auth"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/bankaccount"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/botapi"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/category"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/config"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/database"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/investment"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/transaction"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/user"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/web"
)

func main() {
	cfg := config.Load()

	log.Printf("[INFO] Iniciando Meu Dinheiro API na porta :%s...", cfg.Port)
	log.Printf("[INFO] Database target configurado: %s", database.MaskDatabaseURL(cfg.DatabaseURL))

	// 1. Executa migrações de banco de dados
	log.Printf("[INFO] Executando migrações com Goose...")
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("[FATAL] Erro ao aplicar migrações: %v", err)
	}
	log.Printf("[INFO] Migrações aplicadas com sucesso.")

	// 2. Conecta ao pool de conexões do PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.ConnectPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao conectar ao banco de dados: %v", err)
	}
	defer pool.Close()

	// 3. Inicializa Repositórios
	userRepo := user.NewRepository(pool)
	bankAccountRepo := bankaccount.NewRepository(pool)
	categoryRepo := category.NewRepository(pool)
	transactionRepo := transaction.NewRepository(pool)
	investRepo := investment.NewRepository(pool)

	// 4. Inicializa Serviços
	jwtService := auth.NewJWTService(cfg.JWTSecret)
	authService := auth.NewService(pool, userRepo, jwtService, cfg.BotUsername)
	transactionService := transaction.NewService(transactionRepo, categoryRepo, bankAccountRepo)
	investService := investment.NewService(investRepo)

	// 5. Inicializa Handlers
	authHandler := auth.NewHandler(authService, cfg.InternalAPIKey)
	bankAccountHandler := bankaccount.NewHandler(bankAccountRepo)
	categoryHandler := category.NewHandler(categoryRepo)
	transactionHandler := transaction.NewHandler(transactionService)
	investHandler := investment.NewHandler(investService)
	botAPIHandler := botapi.NewHandler(cfg.InternalAPIKey, userRepo, bankAccountRepo, categoryRepo, transactionRepo, investService, transactionService)

	// 6. Configura Roteamento
	r := web.NewRouter()

	// Healthcheck
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		web.JSON(w, http.StatusOK, map[string]string{"status": "UP"})
	})

	authMiddleware := auth.RequireAuth(jwtService, userRepo, cfg.InternalAPIKey)

	authHandler.RegisterRoutes(r, authMiddleware)
	bankAccountHandler.RegisterRoutes(r, authMiddleware)
	categoryHandler.RegisterRoutes(r, authMiddleware)
	transactionHandler.RegisterRoutes(r, authMiddleware)
	investHandler.RegisterRoutes(r, authMiddleware)
	botAPIHandler.RegisterRoutes(r)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Canal para shutdown gracioso
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[INFO] Servidor HTTP ouvindo em :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] Falha no servidor HTTP: %v", err)
		}
	}()

	<-stop
	log.Printf("[INFO] Encerrando servidor HTTP...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Erro ao encerrar servidor HTTP: %v", err)
	}

	log.Printf("[INFO] Meu Dinheiro API finalizada com sucesso.")
}
