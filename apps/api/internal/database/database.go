package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/migrations"
	"github.com/pressly/goose/v3"
)

// MaskDatabaseURL oculta a senha da URL de conexão para exibição segura nos logs.
func MaskDatabaseURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "[URL inválida]"
	}
	if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(parsed.User.Username(), "****")
		}
	}
	return parsed.String()
}

// ConnectPool inicializa o pool de conexões do pgxpool.
func ConnectPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar databaseURL: %w", err)
	}

	cfg.MaxConns = 15
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar pool de conexões: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("falha ao conectar no banco de dados: %w", err)
	}

	return pool, nil
}

// RunMigrations executa as migrações SQL embutidas via Goose com tentativas de reconexão.
func RunMigrations(databaseURL string) error {
	var db *sql.DB
	var err error

	log.Printf("[INFO] Conectando ao PostgreSQL para migrações em: %s", MaskDatabaseURL(databaseURL))

	// Tenta conectar ao banco por até 30 segundos (10 tentativas com intervalo de 3s)
	for attempt := 1; attempt <= 10; attempt++ {
		db, err = sql.Open("pgx", databaseURL)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				log.Printf("[INFO] Conexão com o banco estabelecida com sucesso na tentativa %d.", attempt)
				break
			} else {
				err = pingErr
				log.Printf("[WARN] Tentativa %d/10 falhou ao conectar (%s): %v", attempt, MaskDatabaseURL(databaseURL), pingErr)
				_ = db.Close()
			}
		} else {
			log.Printf("[WARN] Tentativa %d/10 erro ao inicializar driver: %v", attempt, err)
		}
		if attempt < 10 {
			time.Sleep(3 * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("erro ao conectar ao banco (%s) após 10 tentativas: %w", MaskDatabaseURL(databaseURL), err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("erro ao definir dialeto do goose: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("erro ao aplicar migrações: %w", err)
	}

	return nil
}

// WithTx executa uma função em transação atômica. Se a função retornar erro, faz Rollback; senão, faz Commit.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit da transação: %w", err)
	}

	return nil
}
