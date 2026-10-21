-- Tambahkan kolom promotion_end_at untuk mencatat kapan promosi postingan berakhir
ALTER TABLE business_posts ADD COLUMN IF NOT EXISTS promotion_end_at TIMESTAMPTZ;

-- Sinkronisasi postingan promosi yang sudah ada dengan tanggal berakhir promosi katalognya
UPDATE business_posts bp
SET promotion_end_at = p.end_at
FROM promotions p
WHERE bp.promotion_id = p.promotion_id AND bp.promotion_end_at IS NULL;

-- Berikan default durasi 7 hari bagi postingan promosi standalone yang belum memiliki end date
UPDATE business_posts
SET promotion_end_at = created_at + INTERVAL '7 days'
WHERE is_promotion = true AND promotion_end_at IS NULL;

-- Matikan status promosi untuk postingan yang promosinya sudah berakhir
UPDATE business_posts
SET is_promotion = false, updated_at = NOW()
WHERE is_promotion = true AND (
    (promotion_id IN (SELECT promotion_id FROM promotions WHERE status IN ('ended', 'stopped') OR end_at <= NOW()))
    OR (promotion_end_at IS NOT NULL AND promotion_end_at <= NOW())
);
