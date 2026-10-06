-- ==============================================================================
-- MIGRASI: PEMISAHAN UNIQUE EMAIL & PHONE NUMBER PER KELOMPOK ROLE
-- Deskripsi:
--   Mengizinkan email/nomor HP yang sama digunakan untuk akun Customer dan akun Merchant (Owner/Admin),
--   namun tetap mencegah akun Owner dan Admin berbagi email/nomor HP yang sama.
--
-- Menjamin:
--   1. Akun Customer dan Owner/Admin memiliki account_id terpisah 100%.
--   2. Relasi data (review, wishlist, bookmark, transaksi, signal, bisnis, dll.)
--      tidak pernah bercampur karena terikat ke foreign key account_id.
-- ==============================================================================

DO $$
DECLARE
    r RECORD;
BEGIN
    -- 1. Hapus constraint UNIQUE global lama pada kolom email dan phone_number di tabel accounts
    FOR r IN (
        SELECT tc.constraint_name 
        FROM information_schema.table_constraints tc 
        JOIN information_schema.key_column_usage kcu 
          ON tc.constraint_name = kcu.constraint_name 
         AND tc.table_schema = kcu.table_schema
        WHERE tc.constraint_type = 'UNIQUE' 
          AND tc.table_name = 'accounts' 
          AND kcu.column_name IN ('email', 'phone_number')
    ) LOOP
        RAISE NOTICE 'Dropping constraint %', r.constraint_name;
        EXECUTE 'ALTER TABLE accounts DROP CONSTRAINT IF EXISTS ' || quote_ident(r.constraint_name);
    END LOOP;

    -- 2. Hapus indeks unik lama jika ada
    DROP INDEX IF EXISTS idx_accounts_email;
    DROP INDEX IF EXISTS idx_accounts_phone_number;
    DROP INDEX IF EXISTS accounts_email_key;
    DROP INDEX IF EXISTS accounts_phone_number_key;
    DROP INDEX IF EXISTS uq_accounts_email;
    DROP INDEX IF EXISTS uq_accounts_phone;

    -- 3. Buat Partial Unique Index per kelompok role:

    -- [A] Kelompok Customer: 1 email/nomor unik untuk akun customer
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_email 
        ON accounts (LOWER(email)) 
        WHERE role = 'customer' AND email IS NOT NULL AND email != '';

    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_phone 
        ON accounts (phone_number) 
        WHERE role = 'customer' AND phone_number IS NOT NULL AND phone_number != '';

    -- [B] Kelompok Merchant: 1 email/nomor unik untuk owner dan business_admin
    --     (Owner dan Admin tetap DILARANG berbagi email/nomor yang sama)
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_email 
        ON accounts (LOWER(email)) 
        WHERE role IN ('owner', 'business_admin') AND email IS NOT NULL AND email != '';

    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_merchant_phone 
        ON accounts (phone_number) 
        WHERE role IN ('owner', 'business_admin') AND phone_number IS NOT NULL AND phone_number != '';

    RAISE NOTICE 'Migration accounts unique per role group completed successfully.';
END $$;
