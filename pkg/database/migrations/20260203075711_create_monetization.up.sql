CREATE TABLE ad_campaigns (
    ad_campaign_id BIGSERIAL PRIMARY KEY,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    target_type enum_ad_target,
    target_id BIGINT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    status enum_ad_status DEFAULT 'active',
    budget_limit DECIMAL(15,2)
);

CREATE TABLE transactions (
    transaction_id BIGSERIAL PRIMARY KEY,
    business_id VARCHAR(50) REFERENCES businesses(business_id),
    amount DECIMAL(15,2),
    type VARCHAR(50),
    status enum_payment_status,
    created_at TIMESTAMPTZ DEFAULT NOW()
);