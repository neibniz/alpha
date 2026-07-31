ALTER TABLE "system_logs"
    ADD COLUMN IF NOT EXISTS "ip" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "country_short" varchar(8) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "region" varchar(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "location" varchar(160) NOT NULL DEFAULT '';

ALTER TABLE "subscribe_access_logs"
    ADD COLUMN IF NOT EXISTS "country_short" varchar(8) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "region" varchar(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "location" varchar(160) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS "idx_system_logs_ip_location"
    ON "system_logs" ("ip", "country_short", "region");

CREATE INDEX IF NOT EXISTS "idx_subscribe_access_ip_location"
    ON "subscribe_access_logs" ("client_ip", "country_short", "region");
