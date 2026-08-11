CREATE TABLE IF NOT EXISTS merchant_analytics (
    log_id BIGSERIAL PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL, 
    source_type VARCHAR(50) NOT NULL,      
    searched_tag VARCHAR(100),       
    product_id VARCHAR(50) REFERENCES products(product_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index agar sistem tidak lemot saat menghitung data hari ini
CREATE INDEX IF NOT EXISTS idx_analytics_time ON merchant_analytics(business_id, event_type, created_at);