CREATE TABLE owners (
    account_id VARCHAR(50) PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    ktp_number VARCHAR(50) UNIQUE,
    ktp_photo_url TEXT,
    referral_code TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()   
);