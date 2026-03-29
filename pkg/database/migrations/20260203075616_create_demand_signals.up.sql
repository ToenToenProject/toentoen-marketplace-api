CREATE TABLE demand_signals (
    demand_signal_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(50) REFERENCES users(account_id) ON DELETE CASCADE,
    keyword VARCHAR(255) NOT NULL,
    description TEXT,
    media_urls JSONB, 
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    radius_km INT,
    status enum_signal_status DEFAULT 'open',
    is_fulfilled BOOLEAN DEFAULT false,
    fulfilled_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE demand_responses (
    demand_response_id BIGSERIAL PRIMARY KEY,
    signal_id BIGINT REFERENCES demand_signals(demand_signal_id) ON DELETE CASCADE,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    message TEXT,
    media_urls JSONB,
    is_accepted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);