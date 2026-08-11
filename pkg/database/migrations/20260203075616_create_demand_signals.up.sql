-- 1. Demand Signals
CREATE TABLE IF NOT EXISTS demand_signals (
    signal_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) REFERENCES accounts(account_id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL, -- (e.g., 'bengkel', 'tambal_ban', 'medis')
    description TEXT,
    media_urls JSONB DEFAULT '[]'::jsonb, 
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    radius_km INT DEFAULT 5,
    status VARCHAR(20) DEFAULT 'open', -- 'open', 'processing', 'completed', 'cancelled'
    is_fulfilled BOOLEAN DEFAULT false,
    fulfilled_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Demand Responses (Tawaran/Bantuan dari Owner)
CREATE TABLE IF NOT EXISTS demand_responses (
    response_id VARCHAR(50) PRIMARY KEY,
    signal_id VARCHAR(50) REFERENCES demand_signals(signal_id) ON DELETE CASCADE,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    message TEXT,
    media_urls JSONB DEFAULT '[]'::jsonb,
    is_accepted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);