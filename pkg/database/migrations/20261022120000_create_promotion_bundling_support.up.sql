-- UP Migration: Add bundling promotion support to promotions and create promotion_bundle_items

ALTER TABLE promotions ADD COLUMN IF NOT EXISTS promotion_type VARCHAR(20) NOT NULL DEFAULT 'per_product';
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS package_name VARCHAR(100) NULL;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS package_price NUMERIC(15,2) NULL;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS package_image VARCHAR(500) NULL;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS original_total_price NUMERIC(15,2) NULL;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS savings_amount NUMERIC(15,2) NULL;
ALTER TABLE promotions ADD COLUMN IF NOT EXISTS savings_percentage NUMERIC(5,2) NULL;

CREATE TABLE IF NOT EXISTS promotion_bundle_items (
    promotion_id VARCHAR(50) NOT NULL REFERENCES promotions(promotion_id) ON DELETE CASCADE,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1,
    item_normal_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (promotion_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_bundle_items_product ON promotion_bundle_items(product_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON promotion_bundle_items TO toentoen_app;
