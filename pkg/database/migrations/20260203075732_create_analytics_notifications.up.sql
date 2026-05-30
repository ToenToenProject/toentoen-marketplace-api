CREATE TABLE analytics_business_daily (
    analytics_id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    profile_visits INT DEFAULT 0,
    impression_count INT DEFAULT 0,
    UNIQUE(date, business_id)
);

CREATE TABLE analytics_search_keywords (
    id BIGSERIAL PRIMARY KEY,
    keyword VARCHAR(255),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE notifications (
    notification_id BIGSERIAL PRIMARY KEY,
    recipient_account_id VARCHAR(50) REFERENCES customers(account_id),
    recipient_business_id VARCHAR(50) REFERENCES businesses(business_id),
    title VARCHAR(255),
    message TEXT,
    type enum_notif_type,
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);