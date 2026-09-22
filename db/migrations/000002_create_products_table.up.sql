CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    price NUMERIC(12, 2) NOT NULL,
    description TEXT NULL,
    category VARCHAR(100) NOT NULL,
    images TEXT[] NOT NULL,
    created_by_id BIGINT NOT NULL REFERENCES users(id),
    updated_by_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_category ON products(LOWER(category));
CREATE INDEX IF NOT EXISTS idx_products_created_at_id ON products(created_at DESC, id DESC);
