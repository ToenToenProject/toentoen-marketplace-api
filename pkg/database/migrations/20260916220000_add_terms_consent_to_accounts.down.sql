-- Drop TnC & Privacy Policy consent tracking from accounts table
ALTER TABLE accounts
DROP COLUMN IF EXISTS accepted_terms_at,
DROP COLUMN IF EXISTS terms_version,
DROP COLUMN IF EXISTS accepted_privacy_at,
DROP COLUMN IF EXISTS privacy_version;
