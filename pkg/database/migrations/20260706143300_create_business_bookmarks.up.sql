CREATE TABLE "business_bookmarks" (
    "account_id" varchar NOT NULL,
    "business_id" varchar NOT NULL,
    "created_at" timestamptz DEFAULT (now()),
    PRIMARY KEY ("account_id", "business_id"),
    FOREIGN KEY ("account_id") REFERENCES "accounts" ("account_id") ON DELETE CASCADE,
    FOREIGN KEY ("business_id") REFERENCES "businesses" ("business_id") ON DELETE CASCADE
);
