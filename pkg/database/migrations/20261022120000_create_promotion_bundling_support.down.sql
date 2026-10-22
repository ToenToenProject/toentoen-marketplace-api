-- DOWN Migration: Revert bundling promotion support

DROP TABLE IF EXISTS promotion_bundle_items CASCADE;

ALTER TABLE promotions DROP COLUMN IF EXISTS savings_percentage;
ALTER TABLE promotions DROP COLUMN IF EXISTS savings_amount;
ALTER TABLE promotions DROP COLUMN IF EXISTS original_total_price;
ALTER TABLE promotions DROP COLUMN IF EXISTS package_image;
ALTER TABLE promotions DROP COLUMN IF EXISTS package_price;
ALTER TABLE promotions DROP COLUMN IF EXISTS package_name;
ALTER TABLE promotions DROP COLUMN IF EXISTS promotion_type;
