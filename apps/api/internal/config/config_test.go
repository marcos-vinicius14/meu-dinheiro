package config_test

import (
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/api/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadWithDirectDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://custom_user:custom_pass@custom_host:5432/custom_db?sslmode=require")
	cfg := config.Load()
	assert.Equal(t, "postgres://custom_user:custom_pass@custom_host:5432/custom_db?sslmode=require", cfg.DatabaseURL)
}

func TestLoadReconstructFromIndividualEnvVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_USER", "app_user")
	t.Setenv("POSTGRES_PASSWORD", "secret_pass_123")
	t.Setenv("POSTGRES_HOST", "db-internal")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_DB", "financeiro")
	t.Setenv("POSTGRES_SSLMODE", "disable")

	cfg := config.Load()
	assert.Equal(t, "postgres://app_user:secret_pass_123@db-internal:5433/financeiro?sslmode=disable", cfg.DatabaseURL)
}

func TestLoadReconstructWithSpecialCharsPassword(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_USER", "app_user")
	t.Setenv("POSTGRES_PASSWORD", "pass with space/slash")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_DB", "meu_dinheiro")

	cfg := config.Load()
	assert.Equal(t, "postgres://app_user:pass%20with%20space%2Fslash@postgres:5433/meu_dinheiro?sslmode=disable", cfg.DatabaseURL)
}

func TestLoadEmptyDatabaseVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_DB", "")

	cfg := config.Load()
	assert.Empty(t, cfg.DatabaseURL)
}
