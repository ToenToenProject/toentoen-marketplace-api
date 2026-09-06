-- Add verification timestamps to accounts
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMP;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS phone_verified_at TIMESTAMP;

-- Add KYC fields to owners
ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'unverified';
ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_rejection_reason TEXT;
ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_submitted_at TIMESTAMP;
ALTER TABLE owners ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMP;

-- Add KYC fields to business_admins
ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'unverified';
ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_rejection_reason TEXT;
ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_submitted_at TIMESTAMP;
ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMP;
