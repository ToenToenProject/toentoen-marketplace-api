CREATE TABLE business_admins (
    account_id VARCHAR(50) PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    owner_id VARCHAR(50) REFERENCES owners(account_id) ON DELETE CASCADE NOT NULL,
    ktp_number VARCHAR(50) UNIQUE,
    ktp_photo_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE business_admin_assignments (
    admin_id VARCHAR(50) REFERENCES business_admins(account_id) ON DELETE CASCADE,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    role enum_business_admin_role DEFAULT 'admin' NOT NULL,
    shift_start TIME,
    shift_end TIME,
    is_active BOOLEAN DEFAULT true, 
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (admin_id, business_id) 
);