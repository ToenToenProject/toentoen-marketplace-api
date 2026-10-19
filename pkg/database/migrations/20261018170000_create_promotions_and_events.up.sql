-- UP Migration: Create promotions, promotion_products, promotion_events, and alter business_posts

CREATE TABLE IF NOT EXISTS promotions (
    promotion_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    caption VARCHAR(150) NOT NULL,
    main_product_id VARCHAR(50) NOT NULL REFERENCES products(product_id),
    post_id VARCHAR(50) NULL REFERENCES business_posts(post_id) ON DELETE SET NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    notified_at TIMESTAMPTZ NULL,
    stopped_at TIMESTAMPTZ NULL,
    impression_count INT NOT NULL DEFAULT 0,
    click_count INT NOT NULL DEFAULT 0,
    created_by VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_promotions_business_status ON promotions(business_id, status);
CREATE INDEX IF NOT EXISTS idx_promotions_worker ON promotions(status, start_at, end_at);

CREATE TABLE IF NOT EXISTS promotion_products (
    promotion_id VARCHAR(50) NOT NULL REFERENCES promotions(promotion_id) ON DELETE CASCADE,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    discount_source VARCHAR(20) NOT NULL DEFAULT 'promotion',
    discount_type VARCHAR(20) NOT NULL,
    discount NUMERIC(15,2) NOT NULL DEFAULT 0,
    max_discount NUMERIC(15,2) NULL,
    had_previous_discount BOOLEAN NOT NULL DEFAULT FALSE,
    prev_discount_type VARCHAR(20) NULL,
    prev_discount NUMERIC(15,2) NULL,
    prev_max_discount NUMERIC(15,2) NULL,
    prev_discount_start_at TIMESTAMPTZ NULL,
    prev_discount_end_at TIMESTAMPTZ NULL,
    snapshot_original_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    snapshot_final_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (promotion_id, product_id)
);

CREATE TABLE IF NOT EXISTS promotion_events (
    event_id VARCHAR(50) PRIMARY KEY,
    promotion_id VARCHAR(50) NOT NULL REFERENCES promotions(promotion_id) ON DELETE CASCADE,
    product_id VARCHAR(50) NULL,
    account_id VARCHAR(50) NULL,
    event_type VARCHAR(20) NOT NULL,
    source VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_promotion_events_query ON promotion_events(promotion_id, event_type);

ALTER TABLE business_posts ADD COLUMN IF NOT EXISTS is_promotion BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE business_posts ADD COLUMN IF NOT EXISTS promotion_id VARCHAR(50) NULL REFERENCES promotions(promotion_id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_business_posts_promotion ON business_posts(promotion_id) WHERE is_promotion = TRUE;
