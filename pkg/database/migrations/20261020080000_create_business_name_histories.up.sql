CREATE TABLE IF NOT EXISTS business_name_histories (
    history_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL,
    old_name VARCHAR(150) NOT NULL,
    new_name VARCHAR(150) NOT NULL,
    changed_by VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_business_name_histories_business FOREIGN KEY (business_id) REFERENCES businesses(business_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_business_name_histories_biz_id ON business_name_histories(business_id);
CREATE INDEX IF NOT EXISTS idx_business_name_histories_created_at ON business_name_histories(created_at);
