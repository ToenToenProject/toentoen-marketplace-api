DROP TABLE IF EXISTS product_comment_reports;
DROP TABLE IF EXISTS product_comment_likes;

ALTER TABLE product_comments
DROP COLUMN IF EXISTS rating,
DROP COLUMN IF EXISTS media_urls;
