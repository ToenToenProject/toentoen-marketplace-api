-- Membuat tabel product_groups
CREATE TABLE IF NOT EXISTS product_groups (
    product_group_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);