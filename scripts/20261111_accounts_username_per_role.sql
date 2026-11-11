-- ==============================================================================
-- MIGRASI: PEMISAHAN UNIQUE USERNAME PER KELOMPOK ROLE
-- Deskripsi:
--   Mengizinkan username yang sama digunakan untuk akun Customer, akun Owner,
--   dan akun Admin (Business Admin).
-- ==============================================================================

DO $$
DECLARE
    r RECORD;
BEGIN
    -- 1. Hapus constraint UNIQUE global lama pada kolom username di tabel accounts
    FOR r IN (
        SELECT tc.constraint_name 
        FROM information_schema.table_constraints tc 
        JOIN information_schema.key_column_usage kcu 
          ON tc.constraint_name = kcu.constraint_name 
         AND tc.table_schema = kcu.table_schema
        WHERE tc.constraint_type = 'UNIQUE' 
          AND tc.table_name = 'accounts' 
          AND kcu.column_name = 'username'
    ) LOOP
        RAISE NOTICE 'Dropping constraint %', r.constraint_name;
        EXECUTE 'ALTER TABLE accounts DROP CONSTRAINT IF EXISTS ' || quote_ident(r.constraint_name);
    END LOOP;

    -- 2. Hapus indeks unik lama jika ada
    DROP INDEX IF EXISTS idx_accounts_username;
    DROP INDEX IF EXISTS accounts_username_key;
    DROP INDEX IF EXISTS uq_accounts_username;

    -- 3. Buat Partial Unique Index per role:
    -- [A] Customer: 1 username unik sesama akun customer
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_customer_username 
        ON accounts (LOWER(username)) 
        WHERE role = 'customer' AND username IS NOT NULL AND username <> '';

    -- [B] Owner: 1 username unik sesama akun owner
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_owner_username 
        ON accounts (LOWER(username)) 
        WHERE role = 'owner' AND username IS NOT NULL AND username <> '';

    -- [C] Business Admin: 1 username unik sesama akun business admin
    CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_admin_username 
        ON accounts (LOWER(username)) 
        WHERE role = 'business_admin' AND username IS NOT NULL AND username <> '';

END $$;
