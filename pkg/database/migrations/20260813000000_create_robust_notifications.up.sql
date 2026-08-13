CREATE TABLE IF NOT EXISTS user_notifications (
    id UUID PRIMARY KEY,
    recipient_account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    actor_ids JSONB DEFAULT '[]', -- Array of account_ids who triggered this (for grouping)
    type VARCHAR(50) NOT NULL, -- e.g., 'product_like', 'product_comment'
    reference_id VARCHAR(50), -- e.g., product_id, comment_id (used to find existing unread notifications for grouping)
    title VARCHAR(255),
    body TEXT,
    data JSONB DEFAULT '{}', -- Flexible payload (e.g., product_image, frontend route)
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_notif_recipient ON user_notifications(recipient_account_id, is_read);
CREATE INDEX idx_user_notif_ref ON user_notifications(recipient_account_id, type, reference_id);
