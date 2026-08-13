-- Ubah nama tabel utama
ALTER TABLE IF EXISTS product_comments RENAME TO product_reviews;
ALTER TABLE IF EXISTS product_comment_likes RENAME TO product_review_likes;
ALTER TABLE IF EXISTS product_comment_reports RENAME TO product_review_reports;

-- Ubah nama kolom yang terkait di product_reviews (dahulu product_comments)
ALTER TABLE IF EXISTS product_reviews RENAME COLUMN comment_id TO review_id;
ALTER TABLE IF EXISTS product_reviews RENAME COLUMN reply_to_comment_id TO reply_to_review_id;

-- Ubah nama kolom foreign key di product_review_likes
ALTER TABLE IF EXISTS product_review_likes RENAME COLUMN comment_id TO review_id;

-- Ubah nama kolom foreign key di product_review_reports
ALTER TABLE IF EXISTS product_review_reports RENAME COLUMN comment_id TO review_id;

-- Hapus kolom comment_count di products
ALTER TABLE IF EXISTS products DROP COLUMN IF EXISTS comment_count;
