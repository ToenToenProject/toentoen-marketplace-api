ALTER TABLE product_comments
ADD COLUMN rating NUMERIC(3, 2),
ADD COLUMN media_urls JSONB DEFAULT '[]'::jsonb;

CREATE TABLE product_comment_likes (
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    comment_id VARCHAR(50) NOT NULL REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, comment_id)
);

CREATE TABLE product_comment_reports (
    report_id SERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    comment_id VARCHAR(50) NOT NULL REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
