-- Migration: Rename product_images to product_preview in products table
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'products' AND column_name = 'product_images'
    ) THEN
        ALTER TABLE products RENAME COLUMN product_images TO product_preview;
    END IF;
END $$;
