-- Hapus kolom manajemen stok fisik (stock_quantity & low_stock_threshold) dari products dan product_variants
ALTER TABLE products DROP COLUMN IF EXISTS stock_quantity;
ALTER TABLE products DROP COLUMN IF EXISTS low_stock_threshold;
ALTER TABLE product_variants DROP COLUMN IF EXISTS stock_quantity;
