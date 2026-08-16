DROP INDEX IF EXISTS idx_assignments_supervisor;
DROP INDEX IF EXISTS idx_assignments_deletion;
DROP INDEX IF EXISTS idx_assignments_status;

ALTER TABLE IF EXISTS business_admin_assignments DROP COLUMN IF EXISTS last_rejected_at;
ALTER TABLE IF EXISTS business_admin_assignments DROP COLUMN IF EXISTS rejection_count;
ALTER TABLE IF EXISTS business_admin_assignments DROP COLUMN IF EXISTS deletion_initiated_by;
ALTER TABLE IF EXISTS business_admin_assignments DROP COLUMN IF EXISTS deletion_scheduled_at;
ALTER TABLE IF EXISTS business_admin_assignments DROP COLUMN IF EXISTS supervisor_id;

ALTER TABLE IF EXISTS business_admins DROP COLUMN IF EXISTS ktp_address;
ALTER TABLE IF EXISTS business_admins DROP COLUMN IF EXISTS ktp_gender;
ALTER TABLE IF EXISTS business_admins DROP COLUMN IF EXISTS ktp_birth_date;
ALTER TABLE IF EXISTS business_admins DROP COLUMN IF EXISTS ktp_birth_place;
ALTER TABLE IF EXISTS business_admins DROP COLUMN IF EXISTS ktp_name;
