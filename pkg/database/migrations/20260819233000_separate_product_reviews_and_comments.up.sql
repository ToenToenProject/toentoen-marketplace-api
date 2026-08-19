-- 1. PASTIKAN TABEL PRODUCT_REVIEWS LENGKAP & BERSIH UNTUK ULASAN BERBINTANG
CREATE TABLE IF NOT EXISTS product_reviews (
    review_id VARCHAR(50) PRIMARY KEY,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    rating DECIMAL(2,1) NOT NULL CHECK (rating >= 1.0 AND rating <= 5.0),
    comment TEXT NOT NULL,
    review_images TEXT DEFAULT '[]',
    review_video TEXT,
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uni_product_account_review UNIQUE(product_id, account_id)
);

-- JIKA TABEL SUDAH ADA DARI SEBELUMNYA, PASTIKAN STRUKTURNYA SESUAI:
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_reviews') THEN
        ALTER TABLE product_reviews ADD COLUMN IF NOT EXISTS comment TEXT;
        ALTER TABLE product_reviews ADD COLUMN IF NOT EXISTS review_images TEXT DEFAULT '[]';
        ALTER TABLE product_reviews ADD COLUMN IF NOT EXISTS review_video TEXT;
        ALTER TABLE product_reviews ADD COLUMN IF NOT EXISTS like_count INT DEFAULT 0;
        ALTER TABLE product_reviews ADD COLUMN IF NOT EXISTS comment_count INT DEFAULT 0;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_reviews' AND column_name = 'content') THEN
            UPDATE product_reviews SET comment = content WHERE (comment IS NULL OR comment = '') AND content IS NOT NULL;
        END IF;
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_reviews' AND column_name = 'media_urls') THEN
            UPDATE product_reviews SET review_images = media_urls::text WHERE (review_images IS NULL OR review_images = '' OR review_images = '[]') AND media_urls IS NOT NULL;
        END IF;
    END IF;
END $$;

-- 2. TABEL LIKE UNTUK REVIEW PRODUK
CREATE TABLE IF NOT EXISTS product_review_likes (
    review_id VARCHAR(50) NOT NULL REFERENCES product_reviews(review_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (review_id, account_id)
);

-- 3. TABEL BALASAN REVIEW PRODUK (CUSTOMER & MERCHANT)
CREATE TABLE IF NOT EXISTS product_review_comments (
    comment_id VARCHAR(50) PRIMARY KEY,
    review_id VARCHAR(50) NOT NULL REFERENCES product_reviews(review_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    reply_to_comment_id VARCHAR(50) REFERENCES product_review_comments(comment_id) ON DELETE CASCADE,
    replied_to_account_id VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL,
    replied_to_account_name VARCHAR(100),
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE product_review_comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 4. TABEL LIKE UNTUK BALASAN REVIEW PRODUK
CREATE TABLE IF NOT EXISTS product_review_comment_likes (
    comment_id VARCHAR(50) NOT NULL REFERENCES product_review_comments(comment_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (comment_id, account_id)
);

-- 5. TABEL PELAPORAN REVIEW PRODUK & BALASANNYA
CREATE TABLE IF NOT EXISTS product_review_reports (
    report_id VARCHAR(50) PRIMARY KEY,
    review_id VARCHAR(50) NOT NULL REFERENCES product_reviews(review_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS product_review_comment_reports (
    report_id VARCHAR(50) PRIMARY KEY,
    comment_id VARCHAR(50) NOT NULL REFERENCES product_review_comments(comment_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. TABEL BARU: PRODUCT_COMMENTS (KHUSUS DISKUSI / TANYA-JAWAB / Q&A PRODUK)
CREATE TABLE IF NOT EXISTS product_comments (
    comment_id VARCHAR(50) PRIMARY KEY,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    reply_to_comment_id VARCHAR(50) REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    replied_to_account_id VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL,
    replied_to_account_name VARCHAR(100),
    like_count INT DEFAULT 0,
    replies_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 7. TABEL LIKE UNTUK KOMENTAR DISKUSI PRODUK
CREATE TABLE IF NOT EXISTS product_comment_likes (
    comment_id VARCHAR(50) NOT NULL REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (comment_id, account_id)
);

-- 8. TABEL PELAPORAN KOMENTAR DISKUSI PRODUK
CREATE TABLE IF NOT EXISTS product_comment_reports (
    report_id VARCHAR(50) PRIMARY KEY,
    comment_id VARCHAR(50) NOT NULL REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- INDEXES FOR MAXIMUM QUERY PERFORMANCE
CREATE INDEX IF NOT EXISTS idx_product_reviews_product_id ON product_reviews(product_id);
CREATE INDEX IF NOT EXISTS idx_product_reviews_business_id ON product_reviews(business_id);
CREATE INDEX IF NOT EXISTS idx_product_reviews_account_id ON product_reviews(account_id);
CREATE INDEX IF NOT EXISTS idx_product_review_comments_review_id ON product_review_comments(review_id);
CREATE INDEX IF NOT EXISTS idx_product_comments_product_id ON product_comments(product_id);
CREATE INDEX IF NOT EXISTS idx_product_comments_business_id ON product_comments(business_id);
