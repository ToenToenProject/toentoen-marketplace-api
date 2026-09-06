-- Revert KYC fields from business_admins
ALTER TABLE business_admins DROP COLUMN IF EXISTS kyc_verified_at;
ALTER TABLE business_admins DROP COLUMN IF EXISTS kyc_submitted_at;
ALTER TABLE business_admins DROP COLUMN IF EXISTS kyc_rejection_reason;
ALTER TABLE business_admins DROP COLUMN IF EXISTS kyc_status;

-- Revert KYC fields from owners
ALTER TABLE owners DROP COLUMN IF EXISTS kyc_verified_at;
ALTER TABLE owners DROP COLUMN IF EXISTS kyc_submitted_at;
ALTER TABLE owners DROP COLUMN IF EXISTS kyc_rejection_reason;
ALTER TABLE owners DROP COLUMN IF EXISTS kyc_status;

-- Revert verification timestamps from accounts
ALTER TABLE accounts DROP COLUMN IF EXISTS phone_verified_at;
ALTER TABLE accounts DROP COLUMN IF EXISTS email_verified_at;
