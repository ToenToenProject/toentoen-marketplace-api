ALTER TABLE businesses ADD COLUMN IF NOT EXISTS is_live BOOLEAN DEFAULT false;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS live_platform VARCHAR(50);
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS live_started_at TIMESTAMPTZ;
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS last_live_notified_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_businesses_is_live ON businesses(is_live) WHERE is_live = true;
