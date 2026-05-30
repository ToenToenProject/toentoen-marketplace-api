CREATE TABLE customers (
    account_id VARCHAR(50) PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    bio TEXT,
    profile_picture_url VARCHAR(255),
    birth_date DATE,
    last_latitude DECIMAL(9,6),
    last_longitude DECIMAL(9,6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);