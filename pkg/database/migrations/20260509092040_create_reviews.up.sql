CREATE TABLE reviews (
    review_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    account_id VARCHAR(50) NOT NULL, -- ID Customer yang mereview
    review_images JSONB DEFAULT '[]'::jsonb,
    review_video TEXT,
    
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    
    reply_comment TEXT, -- Kolom balasan dari owner
    replied_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- Index agar load data review ngebut
CREATE INDEX idx_reviews_business ON reviews(business_id);