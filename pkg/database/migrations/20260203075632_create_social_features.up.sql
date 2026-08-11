-- 1. TABEL POSTINGAN (Diperbaiki)
CREATE TABLE IF NOT EXISTS business_posts (
    post_id VARCHAR(50) PRIMARY KEY, -- Ubah ke VARCHAR agar seragam (Misal: PST-12345)
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE CASCADE,
    content TEXT,
    
    -- Agar validasi 5 Gambar & 1 Video lebih mudah, pisahkan saja kolomnya
    images JSONB DEFAULT '[]'::jsonb, 
    video_url TEXT, 
    
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,       
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);