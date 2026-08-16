ALTER TABLE business_admins
    DROP COLUMN IF EXISTS ktp_rt_rw,
    DROP COLUMN IF EXISTS ktp_village,
    DROP COLUMN IF EXISTS ktp_district,
    DROP COLUMN IF EXISTS ktp_religion,
    DROP COLUMN IF EXISTS ktp_occupation,
    DROP COLUMN IF EXISTS ktp_citizenship;
