-- ============================================================================
-- SKRIP PENGETATAN HAK AKSES DATABASE TOENTOEN (LEAST PRIVILEGE PRINCIPLE)
-- ============================================================================
-- Cara Eksekusi di Server Produksi (EC2):
-- 
-- 1. Buka terminal server EC2 (SSH).
-- 2. Jalankan psql di dalam container PostgreSQL dengan user superuser / root DB:
--      sudo docker exec -it toentoen_postgres psql -U $DB_USER -d $DB_NAME
-- 
-- 3. Atau eksekusi file ini langsung:
--      sudo docker exec -i toentoen_postgres psql -U $DB_USER -d $DB_NAME < scripts/harden_db_permissions.sql
-- ============================================================================

-- 1. Cabut izin pembuatan objek sembarang oleh user publik pada skema public
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

-- 2. Pastikan database tidak dapat diakses sembarangan oleh role anonim
-- (Hanya user yang diberi GRANT CONNECT yang bisa masuk)
REVOKE CONNECT ON DATABASE toentoen_db FROM PUBLIC;

-- 3. Buat role khusus aplikasi 'toentoen_app' jika belum ada
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'toentoen_app') THEN
        -- Ganti password di bawah dengan password acak yang kuat
        CREATE ROLE toentoen_app WITH LOGIN PASSWORD 'ToenToenAppSecurePass2026!';
        RAISE NOTICE 'Role toentoen_app berhasil dibuat.';
    ELSE
        RAISE NOTICE 'Role toentoen_app sudah ada.';
    END IF;
END
$$;

-- 4. Beri izin koneksi ke database toentoen_db
GRANT CONNECT ON DATABASE toentoen_db TO toentoen_app;

-- 5. Beri izin penggunaan skema public
GRANT USAGE ON SCHEMA public TO toentoen_app;

-- 6. Beri HANYA izin DML (Baca & Tulis Data) pada semua tabel yang ada saat ini
-- toentoen_app TIDAK BISA: DROP TABLE, TRUNCATE, ALTER TABLE
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO toentoen_app;

-- 7. Beri izin sequence (diperlukan untuk kolom ID SERIAL / AUTO INCREMENT)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO toentoen_app;

-- 8. Atur Default Privileges untuk tabel-tabel baru yang dibuat oleh migrator di masa depan
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO toentoen_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO toentoen_app;

-- 9. Pastikan toentoen_app secara eksplisit TIDAK BISA membuat tabel baru di skema public
REVOKE CREATE ON SCHEMA public FROM toentoen_app;

-- ============================================================================
-- VERIFIKASI HAK AKSES
-- ============================================================================
SELECT 
    grantee, 
    table_schema, 
    privilege_type 
FROM 
    information_schema.role_table_grants 
WHERE 
    grantee = 'toentoen_app' 
LIMIT 10;
