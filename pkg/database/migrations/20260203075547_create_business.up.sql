CREATE TABLE businesses (
    business_id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES owners(account_id),
    name VARCHAR(255) NOT NULL,
    handle VARCHAR(255) UNIQUE,
    status enum_business_status DEFAULT 'draft',
    subscription_tier enum_subscription DEFAULT 'free',
    logo_url TEXT,
    description TEXT,
    address TEXT,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    is_manually_closed BOOLEAN DEFAULT false,
    phone_contact VARCHAR(50),
    whatsapp_contact VARCHAR(50),
    email_contact VARCHAR(255),
    website_url TEXT,
    social_instagram VARCHAR(255),
    social_tiktok VARCHAR(255),
    average_rating DECIMAL(3,2) DEFAULT 0,
    review_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- Business Tags Pivot
CREATE TABLE business_tags (
    business_id BIGINT REFERENCES businesses(business_id) ON DELETE CASCADE,
    tag_id BIGINT REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, tag_id)
);

-- Business Hours
CREATE TABLE business_hours (
    business_id BIGINT REFERENCES businesses(business_id) ON DELETE CASCADE,
    day_of_week INT,
    open_time TIME,
    close_time TIME,
    is_closed BOOLEAN DEFAULT false
);

CREATE TABLE business_categories (
    business_id BIGINT REFERENCES businesses(business_id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(category_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, category_id)
);
