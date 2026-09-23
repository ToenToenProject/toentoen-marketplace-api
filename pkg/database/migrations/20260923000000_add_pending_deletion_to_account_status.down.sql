ALTER TABLE IF EXISTS accounts DROP COLUMN IF EXISTS scheduled_deletion_at;
ALTER TABLE IF EXISTS accounts DROP COLUMN IF EXISTS deletion_reason;
ALTER TABLE IF EXISTS accounts DROP COLUMN IF EXISTS suspension_reason;
ALTER TABLE IF EXISTS accounts DROP COLUMN IF EXISTS status_updated_by;