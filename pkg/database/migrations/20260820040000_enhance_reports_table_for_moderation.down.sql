DROP INDEX IF EXISTS idx_reports_priority;
DROP INDEX IF EXISTS idx_reports_status;

ALTER TABLE reports
DROP COLUMN IF EXISTS priority,
DROP COLUMN IF EXISTS admin_note,
DROP COLUMN IF EXISTS resolved_at,
DROP COLUMN IF EXISTS resolved_by;
