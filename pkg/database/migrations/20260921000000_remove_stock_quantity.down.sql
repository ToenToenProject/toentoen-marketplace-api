-- Rollback penambahan kolom stok fisik
ALTER TABLE products ADD COLUMN IF NOT EXISTS stock_quantity INT DEFAULT 0;
ALTER TABLE products ADD COLUMN IF NOT EXISTS low_stock_threshold INT DEFAULT 5;
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS stock_quantity INT DEFAULT 0;
