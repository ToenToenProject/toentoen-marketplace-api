-- PostGIS Extension
CREATE EXTENSION IF NOT EXISTS postgis;
ALTER TABLE signals ADD COLUMN IF NOT EXISTS location geometry(Point, 4326);
CREATE INDEX IF NOT EXISTS idx_signals_location ON signals USING GIST(location);

-- Manual Migration for Products
ALTER TABLE products ADD COLUMN IF NOT EXISTS product_origin varchar(50);
ALTER TABLE products ADD COLUMN IF NOT EXISTS product_description varchar(300);

-- Manual Migration for Businesses
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS tags jsonb DEFAULT '[]'::jsonb;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS certificates jsonb DEFAULT '[]'::jsonb;

-- Rename mapping tables safely using anonymous code block
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_categories') THEN
        ALTER TABLE business_categories RENAME TO business_has_categories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_tags') THEN
        ALTER TABLE business_tags RENAME TO business_has_tags;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'categories') THEN
        ALTER TABLE categories RENAME TO business_categories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'tags') THEN
        ALTER TABLE tags RENAME TO business_tags;
    END IF;
END $$;

-- Drop obsolete tables and columns
DROP TABLE IF EXISTS tag_groups CASCADE;
ALTER TABLE business_categories DROP COLUMN IF EXISTS icon_url CASCADE;

-- Ensure parent_id and category_id columns exist (since AutoMigrate used to do this)
ALTER TABLE business_categories ADD COLUMN IF NOT EXISTS parent_id BIGINT;
ALTER TABLE business_tags ADD COLUMN IF NOT EXISTS parent_id BIGINT;
ALTER TABLE business_tags ADD COLUMN IF NOT EXISTS category_id BIGINT;
