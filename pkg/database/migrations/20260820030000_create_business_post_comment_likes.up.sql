CREATE TABLE IF NOT EXISTS business_post_comment_likes (
    comment_id VARCHAR(50) NOT NULL REFERENCES business_post_comments(comment_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (comment_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_business_post_comment_likes_comment ON business_post_comment_likes(comment_id);
CREATE INDEX IF NOT EXISTS idx_business_post_comment_likes_account ON business_post_comment_likes(account_id);
