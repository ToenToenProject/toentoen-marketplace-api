-- Pastikan kolom deleted_at ada di semua tabel interaksi dan ulasan untuk mendukung GORM Soft Delete

ALTER TABLE IF EXISTS product_comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_product_comments_deleted_at ON product_comments(deleted_at);

ALTER TABLE IF EXISTS product_reviews ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_product_reviews_deleted_at ON product_reviews(deleted_at);

ALTER TABLE IF EXISTS reviews ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_reviews_deleted_at ON reviews(deleted_at);

ALTER TABLE IF EXISTS business_post_comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_business_post_comments_deleted_at ON business_post_comments(deleted_at);

ALTER TABLE IF EXISTS product_review_comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_product_review_comments_deleted_at ON product_review_comments(deleted_at);

ALTER TABLE IF EXISTS business_review_comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_business_review_comments_deleted_at ON business_review_comments(deleted_at);
