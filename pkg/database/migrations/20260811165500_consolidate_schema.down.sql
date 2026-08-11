-- Revert columns
ALTER TABLE business_categories DROP COLUMN IF EXISTS parent_id CASCADE;
ALTER TABLE business_tags DROP COLUMN IF EXISTS parent_id CASCADE;
ALTER TABLE business_tags DROP COLUMN IF EXISTS category_id CASCADE;

-- Rename tables back
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_has_categories') THEN
        ALTER TABLE business_has_categories RENAME TO business_categories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_has_tags') THEN
        ALTER TABLE business_has_tags RENAME TO business_tags;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_categories') THEN
        ALTER TABLE business_categories RENAME TO categories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'business_tags') THEN
        ALTER TABLE business_tags RENAME TO tags;
    END IF;
END $$;

-- Manual Migration for Businesses
ALTER TABLE businesses DROP COLUMN IF EXISTS tags;
ALTER TABLE businesses DROP COLUMN IF EXISTS certificates;

-- Manual Migration for Products
ALTER TABLE products DROP COLUMN IF EXISTS product_origin;
ALTER TABLE products DROP COLUMN IF EXISTS product_description;

-- PostGIS Reversion
DROP INDEX IF EXISTS idx_signals_location;
ALTER TABLE signals DROP COLUMN IF EXISTS location;
