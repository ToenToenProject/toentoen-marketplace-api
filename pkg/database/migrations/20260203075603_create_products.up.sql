CREATE TABLE products (
    product_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    product_group_id VARCHAR(50) REFERENCES product_groups(product_group_id) ON DELETE SET NULL,
    
    -- Informasi Dasar
    product_images JSONB DEFAULT '[]'::jsonb,
    product_name VARCHAR(255) NOT NULL,
    product_description TEXT,
    
    -- Harga & Diskon (TERBARU: Penjadwalan Diskon)
    price DECIMAL(15,2) NOT NULL,
    discount DECIMAL(15,2) DEFAULT 0,
    discount_start_at TIMESTAMPTZ, -- Waktu mulai diskon (bisa NULL)
    discount_end_at TIMESTAMPTZ,   -- Waktu selesai diskon (bisa NULL)
    
    -- Manajemen Stok (TERBARU: Kuantitas & Alert)
    stock_type VARCHAR(50) DEFAULT 'in_stock', -- ENUM/String: in_stock, out_of_stock, limited
    stock_quantity INT DEFAULT 0,              -- Jumlah stok fisik
    low_stock_threshold INT DEFAULT 5,         -- Ambang batas untuk alert stok menipis
    
    -- Klasifikasi & Tampilan
    tags JSONB DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    
    -- Analitik (Denormalisasi untuk performa baca)
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    view_count INT DEFAULT 0,
    sold_count INT DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0,
    review_count INT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE product_variants (
    variant_id VARCHAR(50) PRIMARY KEY,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    
    sku VARCHAR(100),                            -- Stock Keeping Unit (opsional untuk merchant)
    variant_name VARCHAR(255) NOT NULL,          -- Contoh: "Large", "Pandan Dingin", "Size M"
    additional_price DECIMAL(15,2) DEFAULT 0,    -- Harga tambahan (jika ukuran Large nambah Rp5000)
    stock_quantity INT DEFAULT 0,                -- Stok spesifik untuk varian ini
    
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);