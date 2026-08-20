ALTER TABLE reports 
ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'medium',
ADD COLUMN IF NOT EXISTS admin_note TEXT,
ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS resolved_by VARCHAR(50);

CREATE INDEX IF NOT EXISTS idx_reports_priority ON reports(priority);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
