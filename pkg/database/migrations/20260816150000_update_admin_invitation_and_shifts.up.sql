-- 1. Tambah kolom invitation_code ke tabel business_admins
ALTER TABLE business_admins
    ADD COLUMN IF NOT EXISTS invitation_code VARCHAR(20) UNIQUE;

-- 2. Hapus kolom shift sederhana di penugasan
ALTER TABLE business_admin_assignments
    DROP COLUMN IF NOT EXISTS shift_start,
    DROP COLUMN IF NOT EXISTS shift_end;

-- 3. Buat tabel shift multi-hari
CREATE TABLE IF NOT EXISTS business_admin_shifts (
    assignment_id VARCHAR(50) REFERENCES business_admin_assignments(assignment_id) ON DELETE CASCADE,
    day_of_week INT NOT NULL,
    start_time TIME,
    end_time TIME,
    PRIMARY KEY (assignment_id, day_of_week)
);
