-- 1. Tambahkan kolom KYC pada tabel business_admins jika belum ada
ALTER TABLE IF EXISTS business_admins ADD COLUMN IF NOT EXISTS ktp_name VARCHAR(255);
ALTER TABLE IF EXISTS business_admins ADD COLUMN IF NOT EXISTS ktp_birth_place VARCHAR(255);
ALTER TABLE IF EXISTS business_admins ADD COLUMN IF NOT EXISTS ktp_birth_date DATE;
ALTER TABLE IF EXISTS business_admins ADD COLUMN IF NOT EXISTS ktp_gender VARCHAR(50);
ALTER TABLE IF EXISTS business_admins ADD COLUMN IF NOT EXISTS ktp_address TEXT;

-- 2. Tambahkan kolom hierarki, grace period, dan cooldown pada business_admin_assignments
ALTER TABLE IF EXISTS business_admin_assignments ADD COLUMN IF NOT EXISTS supervisor_id VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL;
ALTER TABLE IF EXISTS business_admin_assignments ADD COLUMN IF NOT EXISTS deletion_scheduled_at TIMESTAMPTZ;
ALTER TABLE IF EXISTS business_admin_assignments ADD COLUMN IF NOT EXISTS deletion_initiated_by VARCHAR(50);
ALTER TABLE IF EXISTS business_admin_assignments ADD COLUMN IF NOT EXISTS rejection_count INT DEFAULT 0;
ALTER TABLE IF EXISTS business_admin_assignments ADD COLUMN IF NOT EXISTS last_rejected_at TIMESTAMPTZ;

-- 3. Indeks untuk query performa
CREATE INDEX IF NOT EXISTS idx_assignments_status ON business_admin_assignments(status);
CREATE INDEX IF NOT EXISTS idx_assignments_deletion ON business_admin_assignments(deletion_scheduled_at) WHERE status = 'scheduled_for_deletion';
CREATE INDEX IF NOT EXISTS idx_assignments_supervisor ON business_admin_assignments(business_id, supervisor_id);
