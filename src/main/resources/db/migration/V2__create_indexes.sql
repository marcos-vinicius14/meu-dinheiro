CREATE INDEX IF NOT EXISTS idx_bank_accounts_user_id
    ON tb_bank_accounts (user_id);

CREATE INDEX IF NOT EXISTS idx_categories_user_id
    ON tb_categories (user_id);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date
    ON tb_transactions (user_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_bank_account_date
    ON tb_transactions (bank_account_id, date DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_category_id
    ON tb_transactions (category_id);
