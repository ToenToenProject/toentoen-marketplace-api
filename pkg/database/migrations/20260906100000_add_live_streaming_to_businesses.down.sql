DROP INDEX IF EXISTS idx_businesses_is_live;
ALTER TABLE businesses DROP COLUMN IF EXISTS last_live_notified_at;
ALTER TABLE businesses DROP COLUMN IF EXISTS live_started_at;
ALTER TABLE businesses DROP COLUMN IF EXISTS live_platform;
ALTER TABLE businesses DROP COLUMN IF EXISTS is_live;
