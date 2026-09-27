package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/migrations"
	"github.com/pressly/goose/v3"
)

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

// RunMigrations executa as migrações SQL embutidas via Goose.
func RunMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("erro ao abrir conexão para migrações: %w", err)
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
