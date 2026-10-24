-- Add like_count column to businesses table
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS like_count INT DEFAULT 0;

-- Create business_likes table
CREATE TABLE IF NOT EXISTS "business_likes" (
    "account_id" varchar NOT NULL,
    "business_id" varchar NOT NULL,
    "created_at" timestamptz DEFAULT (now()),
    PRIMARY KEY ("account_id", "business_id"),
    FOREIGN KEY ("account_id") REFERENCES "accounts" ("account_id") ON DELETE CASCADE,
    FOREIGN KEY ("business_id") REFERENCES "businesses" ("business_id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_business_likes_business_id ON business_likes (business_id);
