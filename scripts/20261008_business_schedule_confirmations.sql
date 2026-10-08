-- ==============================================================================
-- MIGRASI: KONFIRMASI JADWAL OPERASIONAL & ATURAN AKTIVASI TOKO (HARIAN)
-- Deskripsi:
--   Menambahkan kolom confirmed_open_date dan confirmed_overtime_date pada
--   tabel businesses untuk mendukung sistem reminder konfirmasi pembukaan toko
--   dan konfirmasi lembur (overtime).
--
-- Aturan Bisnis:
--   1. Default toko selalu "Tutup" (is_manually_closed = true), merchant harus mengaktifkannya.
--   2. Jika toko memiliki jadwal, toko hanya buka jika merchant mengonfirmasi pembukaan
--      hari tersebut (confirmed_open_date = CURRENT_DATE).
--   3. Jika toko ingin buka melewati jam tutup reguler (lembur / bersambung tengah malam),
--      harus dikonfirmasi lembur (confirmed_overtime_date = CURRENT_DATE).
--   4. Jam 00:00 WIB tengah malam, worker harian akan mereset status toko menjadi Tutup.
-- ==============================================================================

DO $$
BEGIN
    -- 1. Tambah kolom confirmed_open_date
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'businesses' AND column_name = 'confirmed_open_date'
    ) THEN
        ALTER TABLE businesses ADD COLUMN confirmed_open_date DATE;
        RAISE NOTICE 'Kolom confirmed_open_date berhasil ditambahkan ke businesses.';
    END IF;

    -- 2. Tambah kolom confirmed_overtime_date
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'businesses' AND column_name = 'confirmed_overtime_date'
    ) THEN
        ALTER TABLE businesses ADD COLUMN confirmed_overtime_date DATE;
        RAISE NOTICE 'Kolom confirmed_overtime_date berhasil ditambahkan ke businesses.';
    END IF;
END $$;
