-- ==============================================================================
-- MIGRASI: PEMISAHAN UNIQUE EMAIL & PHONE NUMBER UNTUK SETIAP PERAN (ROLE)
-- Deskripsi:
--   Mengizinkan 1 email atau nomor telepon yang sama digunakan untuk akun
--   Customer, Owner, dan Business Admin secara mandiri.
--   Masing-masing akun tetap memiliki account_id terpisah 100%.
-- ==============================================================================

DO $$
BEGIN
    -- 1. Hapus partial unique index merchant lama yang menggabungkan owner & business_admin
    DROP INDEX IF EXISTS uq_accounts_merchant_email;
    DROP INDEX IF EXISTS uq_accounts_merchant_phone;

    -- 2. Pastikan partial unique index untuk customer ada
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_email 
        ON accounts (LOWER(email)) 
        WHERE role = 'customer' AND email IS NOT NULL AND email != '';

    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_phone 
        ON accounts (phone_number) 
        WHERE role = 'customer' AND phone_number IS NOT NULL AND phone_number != '';

    -- 3. Buat partial unique index untuk role owner
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_owner_email 
        ON accounts (LOWER(email)) 
        WHERE role = 'owner' AND email IS NOT NULL AND email != '';

    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_owner_phone 
        ON accounts (phone_number) 
        WHERE role = 'owner' AND phone_number IS NOT NULL AND phone_number != '';

    -- 4. Buat partial unique index untuk role business_admin
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_admin_email 
        ON accounts (LOWER(email)) 
        WHERE role = 'business_admin' AND email IS NOT NULL AND email != '';

    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_admin_phone 
        ON accounts (phone_number) 
        WHERE role = 'business_admin' AND phone_number IS NOT NULL AND phone_number != '';

    RAISE NOTICE 'Migration separate accounts unique per role completed successfully.';
END $$;
