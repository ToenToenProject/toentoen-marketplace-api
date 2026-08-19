-- 1. Pastikan kolom like_count dan comment_count ada di tabel reviews
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS like_count INT DEFAULT 0;
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS comment_count INT DEFAULT 0;

-- 2. Tabel Like untuk Review Bisnis
CREATE TABLE IF NOT EXISTS business_review_likes (
    review_id VARCHAR(50) NOT NULL REFERENCES reviews(review_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (review_id, account_id)
);

-- 3. Tabel Balasan / Komentar untuk Review Bisnis (Bisa dibalas oleh customer & merchant)
CREATE TABLE IF NOT EXISTS business_review_comments (
    comment_id VARCHAR(50) PRIMARY KEY,
    review_id VARCHAR(50) NOT NULL REFERENCES reviews(review_id) ON DELETE CASCADE,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    reply_to_comment_id VARCHAR(50) REFERENCES business_review_comments(comment_id) ON DELETE CASCADE,
    replied_to_account_id VARCHAR(50),
    replied_to_account_name VARCHAR(255),
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_biz_review_comments_review ON business_review_comments(review_id);
CREATE INDEX IF NOT EXISTS idx_biz_review_comments_biz ON business_review_comments(business_id);

-- 4. Tabel Like untuk Balasan Komentar Review Bisnis
CREATE TABLE IF NOT EXISTS business_review_comment_likes (
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    comment_id VARCHAR(50) NOT NULL REFERENCES business_review_comments(comment_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (account_id, comment_id)
);

-- 5. Tabel Laporan untuk Review Bisnis & Komentarnya
CREATE TABLE IF NOT EXISTS business_review_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    review_id VARCHAR(50) NOT NULL REFERENCES reviews(review_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS business_review_comment_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    comment_id VARCHAR(50) NOT NULL REFERENCES business_review_comments(comment_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. Tabel Laporan Komentar Postingan
CREATE TABLE IF NOT EXISTS business_post_comment_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    comment_id VARCHAR(50) NOT NULL REFERENCES business_post_comments(comment_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
