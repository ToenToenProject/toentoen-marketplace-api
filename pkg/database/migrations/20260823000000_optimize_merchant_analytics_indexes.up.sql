-- Optimization: Functional and composite indexes for merchant_analytics fast aggregation
CREATE INDEX IF NOT EXISTS idx_analytics_date_lookup 
ON merchant_analytics(business_id, event_type, (DATE(created_at AT TIME ZONE 'Asia/Jakarta')));

CREATE INDEX IF NOT EXISTS idx_analytics_explore_source 
ON merchant_analytics(business_id, event_type, source_type);

