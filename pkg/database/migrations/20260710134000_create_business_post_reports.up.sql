CREATE TABLE IF NOT EXISTS business_post_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    post_id VARCHAR(50) NOT NULL REFERENCES business_posts(post_id) ON DELETE CASCADE,
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business_post_report_evidences (
    evidence_id SERIAL PRIMARY KEY,
    report_id INT NOT NULL REFERENCES business_post_reports(report_id) ON DELETE CASCADE,
    file_url VARCHAR(255) NOT NULL
);
