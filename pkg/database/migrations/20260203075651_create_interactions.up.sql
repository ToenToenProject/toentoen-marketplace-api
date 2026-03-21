CREATE TABLE bookmarks (
    bookmark_id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES users(account_id) ON DELETE CASCADE,
    target_type enum_bookmark_target NOT NULL,
    target_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(account_id, target_type, target_id)
);

CREATE TABLE likes (
    like_id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES users(account_id) ON DELETE CASCADE,
    business_id BIGINT REFERENCES businesses(business_id), -- Optional context
    target_type enum_like_target NOT NULL,
    target_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(account_id, target_type, target_id)
);