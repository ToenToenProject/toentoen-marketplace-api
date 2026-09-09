-- Tambah kolom foto selfie untuk keperluan verifikasi KYC internal
ALTER TABLE owners ADD COLUMN IF NOT EXISTS selfie_photo_url TEXT;
ALTER TABLE business_admins ADD COLUMN IF NOT EXISTS selfie_photo_url TEXT;
