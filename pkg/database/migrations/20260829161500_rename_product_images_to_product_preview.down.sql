-- Migration DOWN: Revert product_preview to product_images in products table
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'products' AND column_name = 'product_preview'
    ) THEN
        ALTER TABLE products RENAME COLUMN product_preview TO product_images;
    END IF;
END $$;
