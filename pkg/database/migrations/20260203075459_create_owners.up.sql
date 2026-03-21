CREATE TABLE owners (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    ktp_number VARCHAR(50) UNIQUE,
    ktp_photo_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);