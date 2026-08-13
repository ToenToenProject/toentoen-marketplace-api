-- Rollback penghapusan kolom comment_count
ALTER TABLE IF EXISTS products ADD COLUMN IF NOT EXISTS comment_count integer DEFAULT 0;

-- Rollback nama kolom foreign key di product_review_reports
ALTER TABLE IF EXISTS product_review_reports RENAME COLUMN review_id TO comment_id;

-- Rollback nama kolom foreign key di product_review_likes
ALTER TABLE IF EXISTS product_review_likes RENAME COLUMN review_id TO comment_id;

-- Rollback nama kolom yang terkait di product_reviews
ALTER TABLE IF EXISTS product_reviews RENAME COLUMN review_id TO comment_id;
ALTER TABLE IF EXISTS product_reviews RENAME COLUMN reply_to_review_id TO reply_to_comment_id;

-- Rollback nama tabel utama
ALTER TABLE IF EXISTS product_review_reports RENAME TO product_comment_reports;
ALTER TABLE IF EXISTS product_review_likes RENAME TO product_comment_likes;
ALTER TABLE IF EXISTS product_reviews RENAME TO product_comments;
