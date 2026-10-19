-- DOWN Migration: Drop promotions, promotion_products, promotion_events, and revert business_posts

DROP INDEX IF EXISTS idx_business_posts_promotion;
ALTER TABLE business_posts DROP COLUMN IF EXISTS promotion_id;
ALTER TABLE business_posts DROP COLUMN IF EXISTS is_promotion;

DROP TABLE IF EXISTS promotion_events;
DROP TABLE IF EXISTS promotion_products;
DROP TABLE IF EXISTS promotions;
