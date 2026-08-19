CREATE TABLE IF NOT EXISTS reports (
    report_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    target_type VARCHAR(30) NOT NULL, -- 'business', 'product', 'review', 'comment', 'post'
    target_id VARCHAR(50) NOT NULL,
    reason TEXT NOT NULL,
    description TEXT,
    evidence TEXT DEFAULT '[]',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_account_id ON reports(account_id);
CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
