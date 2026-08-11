ALTER TABLE "businesses" ADD COLUMN IF NOT EXISTS "location" geometry(Point, 4326);
