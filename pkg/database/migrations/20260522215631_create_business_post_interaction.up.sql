CREATE TABLE business_post_likes (
    account_id VARCHAR(50) NOT NULL,
    post_id VARCHAR(50) NOT NULL REFERENCES business_posts(post_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (account_id, post_id)
);

-- Tabel Business Post Comments
CREATE TABLE business_post_comments (
    comment_id VARCHAR(50) PRIMARY KEY,
    post_id VARCHAR(50) NOT NULL, -- Tambahkan REFERENCES business_posts(post_id) ON DELETE CASCADE jika tabelnya sudah ada
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    
    -- Kunci Nesting Level 1
    reply_to_comment_id VARCHAR(50) REFERENCES business_post_comments(comment_id) ON DELETE CASCADE,
    
    -- Fitur Mention/Tagging
    replied_to_account_id VARCHAR(50),
    replied_to_account_name VARCHAR(255),
    
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);