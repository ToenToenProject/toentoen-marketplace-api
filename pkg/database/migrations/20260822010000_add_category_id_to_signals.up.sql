-- Tambahkan kolom category_id pada tabel signals untuk relasi formal ke business_categories
ALTER TABLE IF EXISTS signals ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES business_categories(category_id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_signals_category_id ON signals(category_id);
