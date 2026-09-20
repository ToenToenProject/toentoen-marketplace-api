ALTER TABLE businesses ADD COLUMN IF NOT EXISTS custom_tags JSONB DEFAULT '[]'::jsonb;
