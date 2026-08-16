-- Tambah kolom KTP lengkap ke tabel business_admins agar setara dengan owners
ALTER TABLE business_admins
    ADD COLUMN IF NOT EXISTS ktp_rt_rw        VARCHAR(20),
    ADD COLUMN IF NOT EXISTS ktp_village      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ktp_district     VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ktp_religion     VARCHAR(50),
    ADD COLUMN IF NOT EXISTS ktp_occupation   VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ktp_citizenship  VARCHAR(50);
