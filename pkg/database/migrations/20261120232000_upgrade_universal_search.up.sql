-- 1. Pemisahan Tipe Tag (Facility vs Service vs Characteristic) pada business_tags
ALTER TABLE business_tags
    ADD COLUMN IF NOT EXISTS tag_type VARCHAR(30) NOT NULL DEFAULT 'characteristic';

CREATE INDEX IF NOT EXISTS idx_business_tags_tag_type ON business_tags (tag_type);

-- Auto-klasifikasi tag yang sudah ada berdasarkan nama
UPDATE business_tags
SET tag_type = 'facility'
WHERE tag_type = 'characteristic'
  AND (
      LOWER(name) LIKE '%wifi%' OR
      LOWER(name) LIKE '%wi-fi%' OR
      LOWER(name) LIKE '%parkir%' OR
      LOWER(name) LIKE '%toilet%' OR
      LOWER(name) LIKE '%mushola%' OR
      LOWER(name) LIKE '%musholla%' OR
      LOWER(name) LIKE '%ac%' OR
      LOWER(name) LIKE '%ruang tunggu%' OR
      LOWER(name) LIKE '%qris%' OR
      LOWER(name) LIKE '%colokan%' OR
      LOWER(name) LIKE '%stop kontak%' OR
      LOWER(name) LIKE '%smoking%' OR
      LOWER(name) LIKE '%merokok%' OR
      LOWER(name) LIKE '%indoor%' OR
      LOWER(name) LIKE '%outdoor%'
  );

UPDATE business_tags
SET tag_type = 'service'
WHERE tag_type = 'characteristic'
  AND (
      LOWER(name) LIKE '%servis%' OR
      LOWER(name) LIKE '%service%' OR
      LOWER(name) LIKE '%ganti oli%' OR
      LOWER(name) LIKE '%cuci%' OR
      LOWER(name) LIKE '%potong rambut%' OR
      LOWER(name) LIKE '%cukur%' OR
      LOWER(name) LIKE '%delivery%' OR
      LOWER(name) LIKE '%pesan antar%' OR
      LOWER(name) LIKE '%reparasi%' OR
      LOWER(name) LIKE '%perbaikan%'
  );

-- 2. Tambah klasifikasi offering_type pada tabel products (MENU_ITEM, RETAIL_PRODUCT, SERVICE, ROOM_TYPE, RENTAL_ITEM, OTHER)
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS offering_type VARCHAR(30) NOT NULL DEFAULT 'OTHER';

CREATE INDEX IF NOT EXISTS idx_products_offering_type ON products (offering_type);

-- 3. Tabel kamus sinonim dinamis (search_synonyms)
CREATE TABLE IF NOT EXISTS search_synonyms (
    synonym_id BIGSERIAL PRIMARY KEY,
    term VARCHAR(120) NOT NULL UNIQUE,
    canonical_term VARCHAR(120) NOT NULL,
    synonyms JSONB NOT NULL DEFAULT '[]'::jsonb,
    dimension VARCHAR(30) NOT NULL DEFAULT 'general',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_synonyms_dimension ON search_synonyms (dimension);
CREATE INDEX IF NOT EXISTS idx_search_synonyms_active ON search_synonyms (is_active);

-- Seed sinonim inti Bahasa Indonesia & layanan lokal (idempotent)
INSERT INTO search_synonyms (term, canonical_term, synonyms, dimension) VALUES
    ('montir', 'Bengkel Mobil', '["bengkel mobil", "bengkel", "mekanik", "servis mobil"]'::jsonb, 'category'),
    ('mekanik', 'Bengkel Mobil', '["bengkel mobil", "bengkel", "montir", "servis mobil"]'::jsonb, 'category'),
    ('bengkel mtr', 'Bengkel Motor', '["bengkel motor", "servis motor"]'::jsonb, 'category'),
    ('ganti oli', 'Ganti Oli', '["ganti oli", "oli mesin", "servis oli", "oil change"]'::jsonb, 'service'),
    ('servis ac', 'Servis AC', '["servis ac", "service ac", "cuci ac", "perbaikan ac", "ac mobil"]'::jsonb, 'service'),
    ('pangkas rambut', 'Barbershop', '["barbershop", "potong rambut", "cukur", "salon", "haircut"]'::jsonb, 'service'),
    ('barbershop', 'Barbershop', '["pangkas rambut", "potong rambut", "cukur", "haircut"]'::jsonb, 'category'),
    ('ayam bakar', 'Ayam Bakar', '["ayam bakar", "ayam panggang", "grilled chicken"]'::jsonb, 'offering'),
    ('ayam panggang', 'Ayam Bakar', '["ayam bakar", "grilled chicken"]'::jsonb, 'offering'),
    ('tempat ngopi', 'Kafe', '["kafe", "cafe", "coffee shop", "warkop", "kedai kopi"]'::jsonb, 'category'),
    ('wifi', 'Wi-Fi', '["wi-fi", "wifi", "internet gratis", "free wifi"]'::jsonb, 'facility'),
    ('wi-fi', 'Wi-Fi', '["wifi", "internet gratis", "free wifi"]'::jsonb, 'facility'),
    ('parkir', 'Parkir', '["parkir luas", "tempat parkir", "parking"]'::jsonb, 'facility'),
    ('ac', 'Ber-AC', '["ber-ac", "ruangan ac", "air conditioning", "dingin"]'::jsonb, 'facility')
ON CONFLICT (term) DO NOTHING;

-- 4. Tabel Search Analytics & Zero-Result Tracking (search_logs)
CREATE TABLE IF NOT EXISTS search_logs (
    log_id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64),
    raw_query TEXT NOT NULL,
    clean_query TEXT,
    parsed_intent JSONB DEFAULT '{}'::jsonb,
    applied_filters JSONB DEFAULT '{}'::jsonb,
    result_count INT NOT NULL DEFAULT 0,
    is_zero_result BOOLEAN NOT NULL DEFAULT FALSE,
    clicked_business_id VARCHAR(64),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_logs_created_at ON search_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_logs_zero_result ON search_logs (is_zero_result) WHERE is_zero_result = TRUE;
CREATE INDEX IF NOT EXISTS idx_search_logs_raw_query ON search_logs (raw_query);
