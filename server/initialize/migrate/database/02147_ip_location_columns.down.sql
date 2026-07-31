DROP INDEX IF EXISTS "idx_subscribe_access_ip_location";
DROP INDEX IF EXISTS "idx_system_logs_ip_location";

ALTER TABLE "subscribe_access_logs"
    DROP COLUMN IF EXISTS "location",
    DROP COLUMN IF EXISTS "region",
    DROP COLUMN IF EXISTS "country_short";

ALTER TABLE "system_logs"
    DROP COLUMN IF EXISTS "location",
    DROP COLUMN IF EXISTS "region",
    DROP COLUMN IF EXISTS "country_short",
    DROP COLUMN IF EXISTS "ip";
