-- Migration: Create Business Transfer Requests Table
CREATE TABLE IF NOT EXISTS business_transfer_requests (
    transfer_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    current_owner_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    target_account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    target_identifier VARCHAR(150) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending_acceptance',
    otp_verified_at TIMESTAMP WITH TIME ZONE,
    accepted_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_btr_business_id ON business_transfer_requests(business_id);
CREATE INDEX IF NOT EXISTS idx_btr_current_owner ON business_transfer_requests(current_owner_id);
CREATE INDEX IF NOT EXISTS idx_btr_target_account ON business_transfer_requests(target_account_id);
CREATE INDEX IF NOT EXISTS idx_btr_status ON business_transfer_requests(status);
