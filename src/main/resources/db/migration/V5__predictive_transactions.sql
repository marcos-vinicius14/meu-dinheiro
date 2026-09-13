-- Categorias: flag de despesa flexível
ALTER TABLE tb_categories
    ADD COLUMN IF NOT EXISTS is_flexible BOOLEAN NOT NULL DEFAULT false;

-- Transações: novo modelo preditivo
ALTER TABLE tb_transactions
    RENAME COLUMN date TO due_date;

ALTER TABLE tb_transactions
    ALTER COLUMN due_date TYPE DATE
    USING due_date::date;

ALTER TABLE tb_transactions
    ALTER COLUMN description TYPE VARCHAR(500);

ALTER TABLE tb_transactions
    ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED';

ALTER TABLE tb_transactions
    ADD COLUMN IF NOT EXISTS payment_date DATE;

ALTER TABLE tb_transactions
    ADD COLUMN IF NOT EXISTS bundle_id UUID;

ALTER TABLE tb_transactions
    ADD COLUMN IF NOT EXISTS installment_number INTEGER;

ALTER TABLE tb_transactions
    ADD COLUMN IF NOT EXISTS total_installments INTEGER;

ALTER TABLE tb_transactions
    ALTER COLUMN bank_account_id DROP NOT NULL;

UPDATE tb_transactions
    SET type = 'FLEXIBLE_EXPENSE'
    WHERE type = 'EXPENSE';

ALTER TABLE tb_transactions
    DROP CONSTRAINT IF EXISTS fk_transaction_bank_account;

ALTER TABLE tb_transactions
    ADD CONSTRAINT fk_transaction_bank_account
        FOREIGN KEY (bank_account_id)
        REFERENCES tb_bank_accounts(id)
        ON DELETE SET NULL;

-- Bundles de compra parcelada
CREATE TABLE IF NOT EXISTS tb_transaction_bundles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    description VARCHAR(255) NOT NULL,
    total_amount NUMERIC(19, 2) NOT NULL,
    total_installments INTEGER NOT NULL,
    first_due_date DATE NOT NULL,

    CONSTRAINT fk_bundle_user
        FOREIGN KEY (user_id)
        REFERENCES tb_users(id)
        ON DELETE CASCADE
);

-- Snapshots do check-in diário (idempotente por usuário/dia)
CREATE TABLE IF NOT EXISTS tb_check_in_snapshots (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    check_in_date DATE NOT NULL,
    s2s_calculated NUMERIC(19, 2) NOT NULL,
    spent_today NUMERIC(19, 2) NOT NULL,
    delta_from_safe_to_spend NUMERIC(19, 2) NOT NULL,
    health_status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_check_in_user
        FOREIGN KEY (user_id)
        REFERENCES tb_users(id)
        ON DELETE CASCADE,

    CONSTRAINT ux_check_in_user_date
        UNIQUE (user_id, check_in_date)
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_status_due
    ON tb_transactions (user_id, status, due_date);

CREATE INDEX IF NOT EXISTS idx_transactions_bundle_id
    ON tb_transactions (bundle_id);
