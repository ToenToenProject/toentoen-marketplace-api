-- =====================================================
-- Migration: Tambah tabel signal_tags
-- Menyimpan tag yang dipilih customer saat buat signal
-- Tag digunakan sebagai relevance booster, BUKAN filter wajib
-- =====================================================

CREATE TABLE IF NOT EXISTS signal_tags (
    signal_id VARCHAR(50) NOT NULL REFERENCES signals(signal_id) ON DELETE CASCADE,
    tag_id    BIGINT      NOT NULL REFERENCES business_tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (signal_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_signal_tags_signal_id ON signal_tags(signal_id);
CREATE INDEX IF NOT EXISTS idx_signal_tags_tag_id    ON signal_tags(tag_id);
