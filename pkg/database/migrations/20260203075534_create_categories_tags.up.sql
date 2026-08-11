-- Categories
CREATE TABLE IF NOT EXISTS categories (
    category_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    icon_url TEXT,
    parent_id INT REFERENCES categories(category_id)
);

-- Tag Groups
CREATE TABLE IF NOT EXISTS tag_groups (
    tag_group_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    tag_id BIGSERIAL PRIMARY KEY,
    tag_group_id INT REFERENCES tag_groups(tag_group_id),
    name VARCHAR(255) NOT NULL
);

-- Customer Preferred Categories (Relation)
CREATE TABLE IF NOT EXISTS customer_preferred_categories (
    account_id VARCHAR(50) REFERENCES customers(account_id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(category_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (account_id, category_id)
);