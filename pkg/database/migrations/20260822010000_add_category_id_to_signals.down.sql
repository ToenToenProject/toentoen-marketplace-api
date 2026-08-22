DROP INDEX IF EXISTS idx_signals_category_id;
ALTER TABLE IF EXISTS signals DROP COLUMN IF EXISTS category_id;
