ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS "client_ip" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "country_short" varchar(8) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "region" varchar(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "location" varchar(160) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS "idx_order_ip_location"
    ON "order" ("client_ip", "country_short", "region");
