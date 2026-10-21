DROP INDEX IF EXISTS uq_accounts_admin_phone;
DROP INDEX IF EXISTS uq_accounts_admin_email;
DROP INDEX IF EXISTS uq_accounts_owner_phone;
DROP INDEX IF EXISTS uq_accounts_owner_email;

CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_email 
    ON accounts (LOWER(email)) 
    WHERE role IN ('owner', 'business_admin') AND email IS NOT NULL AND email != '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_phone 
    ON accounts (phone_number) 
    WHERE role IN ('owner', 'business_admin') AND phone_number IS NOT NULL AND phone_number != '';
