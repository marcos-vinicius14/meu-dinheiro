-- +goose Up
CREATE TABLE IF NOT EXISTS tb_investments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    ticker VARCHAR(12) NOT NULL,
    quantity NUMERIC(15, 4) NOT NULL CHECK (quantity > 0),
    average_price NUMERIC(15, 2) NOT NULL CHECK (average_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ux_investments_user_ticker UNIQUE (user_id, ticker)
);

CREATE INDEX IF NOT EXISTS idx_investments_user_id ON tb_investments (user_id);

ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS target_savings NUMERIC(19, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS flexible_budget_cap NUMERIC(19, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS emergency_fund_target NUMERIC(19, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS emergency_fund_months INTEGER NOT NULL DEFAULT 6;
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS cycle_start_day INTEGER NOT NULL DEFAULT 1 CHECK (cycle_start_day BETWEEN 1 AND 28);

-- +goose Down
ALTER TABLE tb_users DROP COLUMN IF EXISTS cycle_start_day;
ALTER TABLE tb_users DROP COLUMN IF EXISTS emergency_fund_months;
ALTER TABLE tb_users DROP COLUMN IF EXISTS emergency_fund_target;
ALTER TABLE tb_users DROP COLUMN IF EXISTS flexible_budget_cap;
ALTER TABLE tb_users DROP COLUMN IF EXISTS target_savings;
DROP TABLE IF EXISTS tb_investments CASCADE;
