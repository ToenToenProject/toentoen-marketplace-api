CREATE TABLE IF NOT EXISTS businesses (
    business_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) REFERENCES owners(account_id),

    logo_url TEXT,
    business_name VARCHAR(255) NOT NULL,
    handle VARCHAR(255) UNIQUE,
    email VARCHAR(255) NOT NULL,

    status enum_business_status DEFAULT 'draft',
    subscription_tier enum_subscription DEFAULT 'free',

    business_images JSONB DEFAULT '[]' ::jsonb,
    business_details VARCHAR(300),  
    
    province VARCHAR(255),
    city VARCHAR(255),  
    address_detail TEXT,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    
    store_type VARCHAR(50) DEFAULT 'regular',

    is_manually_closed BOOLEAN DEFAULT false,
    phone_contact VARCHAR(50),
    whatsapp_contact VARCHAR(50),
    social_instagram VARCHAR(255),
    social_tiktok VARCHAR(255),
    social_website JSONB DEFAULT '[]' ::jsonb,
    
    average_rating DECIMAL(3,2) DEFAULT 0,
    review_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 3. TABEL RELASI & PIVOT (Tetap Sama)
-- ==========================================

-- Business Hours
CREATE TABLE IF NOT EXISTS business_hours (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    day_of_week INT,
    open_time TIME,
    close_time TIME,
    is_closed BOOLEAN DEFAULT false
);

-- Pivot Categories
CREATE TABLE IF NOT EXISTS business_categories (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(category_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, category_id)
);

-- Pivot Tags
CREATE TABLE IF NOT EXISTS business_tags (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    tag_id BIGINT REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, tag_id)
);