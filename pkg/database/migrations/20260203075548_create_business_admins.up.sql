-- 1. Tabel Profil Admin (Menyimpan data KYC/Pribadi)
CREATE TABLE IF NOT EXISTS business_admins (
    account_id VARCHAR(50) PRIMARY KEY REFERENCES accounts(account_id) ON DELETE CASCADE,
    ktp_number VARCHAR(50) UNIQUE,
    ktp_photo_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Tabel Undangan & Penugasan (Relasi Many-to-Many)
CREATE TABLE IF NOT EXISTS business_admin_assignments (
    assignment_id VARCHAR(50) PRIMARY KEY, -- Menggunakan ID unik (UUID) lebih aman untuk sistem undangan
    admin_id VARCHAR(50) REFERENCES business_admins(account_id) ON DELETE CASCADE,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    
    invited_by VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL, -- Siapa yang mengundang
    
    role VARCHAR(50) DEFAULT 'admin' NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL, -- pending, accepted, rejected, revoked
    
    shift_start TIME,
    shift_end TIME,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Mencegah 1 orang diundang 2 kali ke toko yang sama di waktu bersamaan
    UNIQUE (admin_id, business_id) 
);