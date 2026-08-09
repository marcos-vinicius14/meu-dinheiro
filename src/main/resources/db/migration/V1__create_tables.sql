CREATE TABLE IF NOT EXISTS tb_users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS tb_categories (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    description VARCHAR(255),
    icon VARCHAR(30),

    CONSTRAINT fk_category_user
        FOREIGN KEY (user_id)
        REFERENCES tb_users(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tb_bank_accounts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    initial_balance NUMERIC(19, 2) NOT NULL DEFAULT 0,
    type VARCHAR(30) NOT NULL,

    CONSTRAINT fk_bank_account_user
        FOREIGN KEY (user_id)
        REFERENCES tb_users(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tb_transactions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    bank_account_id UUID NOT NULL,
    category_id UUID NOT NULL,
    description VARCHAR(500),
    value NUMERIC(19, 2) NOT NULL DEFAULT 0,
    type VARCHAR(30) NOT NULL,
    date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transaction_user
        FOREIGN KEY (user_id)
        REFERENCES tb_users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_transaction_bank_account
        FOREIGN KEY (bank_account_id)
        REFERENCES tb_bank_accounts(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_transaction_category
        FOREIGN KEY (category_id)
        REFERENCES tb_categories(id)
        ON DELETE CASCADE
);
