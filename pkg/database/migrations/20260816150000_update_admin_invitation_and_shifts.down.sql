DROP TABLE IF EXISTS business_admin_shifts;

ALTER TABLE business_admin_assignments
    ADD COLUMN IF NOT EXISTS shift_start TIME,
    ADD COLUMN IF NOT EXISTS shift_end TIME;

ALTER TABLE business_admins
    DROP COLUMN IF EXISTS invitation_code;
