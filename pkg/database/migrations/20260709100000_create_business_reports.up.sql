CREATE TABLE IF NOT EXISTS business_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(255) NOT NULL,
    business_id VARCHAR(255) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business_report_evidences (
    evidence_id SERIAL PRIMARY KEY,
    report_id INTEGER NOT NULL REFERENCES business_reports(report_id) ON DELETE CASCADE,
    file_url TEXT NOT NULL
);
