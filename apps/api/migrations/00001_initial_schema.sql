-- +goose Up
CREATE TABLE IF NOT EXISTS tb_users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    telegram_id BIGINT NOT NULL UNIQUE,
    username VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON tb_users(telegram_id);

CREATE TABLE IF NOT EXISTS tb_categories (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    icon VARCHAR(255),
    is_flexible BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_categories_user_id ON tb_categories (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_categories_user_description ON tb_categories (user_id, LOWER(description));

CREATE TABLE IF NOT EXISTS tb_bank_accounts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    initial_balance NUMERIC(19, 2) NOT NULL DEFAULT 0,
    type VARCHAR(30) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bank_accounts_user_id ON tb_bank_accounts (user_id);

CREATE TABLE IF NOT EXISTS tb_transaction_bundles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    total_amount NUMERIC(19, 2) NOT NULL,
    total_installments INTEGER NOT NULL,
    first_due_date DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS tb_transactions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    bank_account_id UUID REFERENCES tb_bank_accounts(id) ON DELETE SET NULL,
    category_id UUID NOT NULL REFERENCES tb_categories(id) ON DELETE CASCADE,
    description VARCHAR(500),
    value NUMERIC(19, 2) NOT NULL DEFAULT 0,
    type VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
    due_date DATE NOT NULL,
    payment_date DATE,
    bundle_id UUID REFERENCES tb_transaction_bundles(id) ON DELETE CASCADE,
    installment_number INTEGER,
    total_installments INTEGER
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_due_date ON tb_transactions (user_id, due_date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_status_due ON tb_transactions (user_id, status, due_date);
CREATE INDEX IF NOT EXISTS idx_transactions_bank_account_date ON tb_transactions (bank_account_id, due_date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON tb_transactions (category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_bundle_id ON tb_transactions (bundle_id);

CREATE TABLE IF NOT EXISTS tb_check_in_snapshots (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
    check_in_date DATE NOT NULL,
    s2s_calculated NUMERIC(19, 2) NOT NULL,
    spent_today NUMERIC(19, 2) NOT NULL,
    delta_from_safe_to_spend NUMERIC(19, 2) NOT NULL,
    health_status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ux_check_in_user_date UNIQUE (user_id, check_in_date)
);

CREATE TABLE IF NOT EXISTS tb_telegram_auth_challenges (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    token VARCHAR(64) NOT NULL UNIQUE,
    user_id UUID REFERENCES tb_users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_telegram_challenges_token ON tb_telegram_auth_challenges (token);

-- +goose Down
DROP TABLE IF EXISTS tb_telegram_auth_challenges CASCADE;
DROP TABLE IF EXISTS tb_check_in_snapshots CASCADE;
DROP TABLE IF EXISTS tb_transactions CASCADE;
DROP TABLE IF EXISTS tb_transaction_bundles CASCADE;
DROP TABLE IF EXISTS tb_bank_accounts CASCADE;
DROP TABLE IF EXISTS tb_categories CASCADE;
DROP TABLE IF EXISTS tb_users CASCADE;
