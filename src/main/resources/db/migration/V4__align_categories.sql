ALTER TABLE tb_categories
    ALTER COLUMN description SET NOT NULL;

ALTER TABLE tb_categories
    ALTER COLUMN icon TYPE VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS ux_categories_user_description
    ON tb_categories (user_id, LOWER(description));
