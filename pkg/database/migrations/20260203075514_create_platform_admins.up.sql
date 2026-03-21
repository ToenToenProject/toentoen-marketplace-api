CREATE TABLE platform_admins (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    role enum_platform_admin_role DEFAULT 'moderator' NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE platform_logs (
    platform_log_id BIGSERIAL PRIMARY KEY,
    admin_id INT REFERENCES platform_admins(account_id) ON DELETE SET NULL, -- Jika admin dihapus, log jangan hilang
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id BIGINT NOT NULL,
    details TEXT,
    ip_address VARCHAR(50),
    user_agent VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_logs_admin ON platform_logs(platform_log_id);
CREATE INDEX idx_logs_target ON platform_logs(target_type, target_id);