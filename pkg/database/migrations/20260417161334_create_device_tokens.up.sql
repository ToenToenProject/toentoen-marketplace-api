CREATE TABLE IF NOT EXISTS device_tokens (
    token_id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    fcm_token TEXT NOT NULL UNIQUE,
    device_type VARCHAR(20), -- contoh: 'android', 'ios', 'web'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Buat index agar pencarian token cepat berdasarkan akun
CREATE INDEX IF NOT EXISTS idx_device_tokens_account ON device_tokens(account_id);