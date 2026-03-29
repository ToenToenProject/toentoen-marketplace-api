-- 1. TABEL UTAMA BUSINESSES
CREATE TABLE businesses (
    business_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) REFERENCES owners(account_id),
    name VARCHAR(255) NOT NULL,
    handle VARCHAR(255) UNIQUE,
    status enum_business_status DEFAULT 'draft',
    subscription_tier enum_subscription DEFAULT 'free',
    logo_url TEXT,
    description TEXT,
    
    -- === PERUBAHAN: Detail Lokasi ===
    province VARCHAR(255),
    city VARCHAR(255),
    address_detail TEXT,
    address TEXT,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    
    -- === PERUBAHAN: Tipe Toko & Tanggal (Seasonal/Regular) ===
    store_type VARCHAR(50) DEFAULT 'regular',
    start_date DATE,
    end_date DATE,

    -- Kontak & Media Sosial
    is_manually_closed BOOLEAN DEFAULT false,
    phone_contact VARCHAR(50),
    whatsapp_contact VARCHAR(50),
    email_contact VARCHAR(255),
    -- website_url DIHAPUS, dipindah ke tabel terpisah
    social_instagram VARCHAR(255),
    social_tiktok VARCHAR(255),
    
    -- Statistik & Timestamps
    average_rating DECIMAL(3,2) DEFAULT 0,
    review_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 2. TABEL BARU: MULTIPLE IMAGES & WEBSITES
-- ==========================================

-- Galeri Foto Bisnis
CREATE TABLE business_images (
    image_id BIGSERIAL PRIMARY KEY,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    image_url TEXT NOT NULL
);

-- Daftar Link Website Bisnis
CREATE TABLE business_websites (
    website_id BIGSERIAL PRIMARY KEY,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    url TEXT NOT NULL
);

-- ==========================================
-- 3. TABEL RELASI & PIVOT (Tetap Sama)
-- ==========================================

-- Business Hours
CREATE TABLE business_hours (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    day_of_week INT,
    open_time TIME,
    close_time TIME,
    is_closed BOOLEAN DEFAULT false
);

-- Pivot Categories
CREATE TABLE business_categories (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    category_id INT REFERENCES categories(category_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, category_id)
);

-- Pivot Tags
CREATE TABLE business_tags (
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    tag_id BIGINT REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, tag_id)
);