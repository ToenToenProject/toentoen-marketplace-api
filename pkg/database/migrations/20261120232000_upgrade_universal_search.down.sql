DROP TABLE IF EXISTS search_logs;
DROP TABLE IF EXISTS search_synonyms;

DROP INDEX IF EXISTS idx_products_offering_type;
ALTER TABLE products DROP COLUMN IF EXISTS offering_type;

DROP INDEX IF EXISTS idx_business_tags_tag_type;
ALTER TABLE business_tags DROP COLUMN IF EXISTS tag_type;
