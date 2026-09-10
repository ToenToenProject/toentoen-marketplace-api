-- 1. User Post Interactions (Feed Logs)
CREATE TABLE IF NOT EXISTS user_post_interactions (
    interaction_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50),
    post_id VARCHAR(50) REFERENCES business_posts(post_id) ON DELETE CASCADE,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    category_id BIGINT REFERENCES business_categories(category_id) ON DELETE SET NULL,
    action_type VARCHAR(50) NOT NULL,
    dwell_time_ms INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_post_interactions_account_id ON user_post_interactions(account_id);
CREATE INDEX IF NOT EXISTS idx_user_post_interactions_post_id ON user_post_interactions(post_id);
CREATE INDEX IF NOT EXISTS idx_user_post_interactions_business_id ON user_post_interactions(business_id);
CREATE INDEX IF NOT EXISTS idx_user_post_interactions_category_id ON user_post_interactions(category_id);
CREATE INDEX IF NOT EXISTS idx_user_post_interactions_action_type ON user_post_interactions(action_type);
CREATE INDEX IF NOT EXISTS idx_user_post_interactions_created_at ON user_post_interactions(created_at);

-- 2. User Category Affinities (Personalization Scores)
CREATE TABLE IF NOT EXISTS user_category_affinities (
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES business_categories(category_id) ON DELETE CASCADE,
    affinity_score NUMERIC(10,2) DEFAULT 1.0,
    interaction_count INT DEFAULT 1,
    last_interacted_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (account_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_user_category_affinities_account_id ON user_category_affinities(account_id);
CREATE INDEX IF NOT EXISTS idx_user_category_affinities_category_id ON user_category_affinities(category_id);
