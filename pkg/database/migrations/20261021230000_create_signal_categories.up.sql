-- UP Migration: Create signal_categories table for multi-category Demand Signals

CREATE TABLE IF NOT EXISTS signal_categories (
    signal_id VARCHAR(50) NOT NULL REFERENCES signals(signal_id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES business_categories(category_id) ON DELETE CASCADE,
    PRIMARY KEY (signal_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_signal_categories_category_id ON signal_categories(category_id);

INSERT INTO signal_categories (signal_id, category_id)
SELECT signal_id, category_id FROM signals
WHERE category_id IS NOT NULL
ON CONFLICT DO NOTHING;

GRANT SELECT, INSERT, UPDATE, DELETE ON signal_categories TO toentoen_app;
