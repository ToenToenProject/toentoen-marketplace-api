DROP INDEX IF EXISTS idx_business_review_comments_deleted_at;
ALTER TABLE IF EXISTS business_review_comments DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_product_review_comments_deleted_at;
ALTER TABLE IF EXISTS product_review_comments DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_business_post_comments_deleted_at;
ALTER TABLE IF EXISTS business_post_comments DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_reviews_deleted_at;
ALTER TABLE IF EXISTS reviews DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_product_reviews_deleted_at;
ALTER TABLE IF EXISTS product_reviews DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_product_comments_deleted_at;
ALTER TABLE IF EXISTS product_comments DROP COLUMN IF EXISTS deleted_at;
