ALTER TABLE businesses ADD COLUMN IF NOT EXISTS hashtags JSONB DEFAULT '[]'::jsonb;
