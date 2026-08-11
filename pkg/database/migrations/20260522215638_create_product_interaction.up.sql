-- Tabel Product Likes
CREATE TABLE IF NOT EXISTS product_likes (
    account_id VARCHAR(50) NOT NULL,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (account_id, product_id)
);

-- Tabel Product Comments
CREATE TABLE IF NOT EXISTS product_comments (
    comment_id VARCHAR(50) PRIMARY KEY,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    
    -- Kunci Nesting Level 1
    reply_to_comment_id VARCHAR(50) REFERENCES product_comments(comment_id) ON DELETE CASCADE,
    
    -- Fitur Mention/Tagging
    replied_to_account_id VARCHAR(50),
    replied_to_account_name VARCHAR(255),
    
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);