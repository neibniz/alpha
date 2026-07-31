DROP INDEX IF EXISTS "idx_order_ip_location";

ALTER TABLE "order"
    DROP COLUMN IF EXISTS "client_ip",
    DROP COLUMN IF EXISTS "country_short",
    DROP COLUMN IF EXISTS "region",
    DROP COLUMN IF EXISTS "location";
