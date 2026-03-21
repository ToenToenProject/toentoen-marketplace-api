-- Business Posts
CREATE TABLE business_posts (
    business_post_id BIGSERIAL PRIMARY KEY,
    business_id BIGINT REFERENCES businesses(business_id) ON DELETE CASCADE,
    content TEXT,
    media_urls JSONB,
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Reviews
CREATE TABLE reviews (
    review_id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES users(account_id),
    business_id BIGINT REFERENCES businesses(business_id),
    product_id BIGINT REFERENCES products(product_id),
    rating INT NOT NULL,
    content TEXT,
    media_urls JSONB,
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Comments (Polymorphic)
CREATE TABLE comments (
    comment_id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES users(account_id) ON DELETE CASCADE,
    business_id BIGINT REFERENCES businesses(business_id), -- Optional context
    target_type enum_comment_target NOT NULL,
    target_id BIGINT NOT NULL,
    content TEXT,
    media_urls JSONB,
    reply_to_comment_id BIGINT REFERENCES comments(comment_id),
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);