ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS "source" varchar(32) NOT NULL DEFAULT 'customer';

ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_source";

ALTER TABLE "order"
    ADD CONSTRAINT "chk_order_source"
    CHECK ("source" IN ('customer', 'admin'));

UPDATE "user_subscribe"
SET "traffic" = GREATEST(COALESCE("traffic", 0), 0),
    "download" = GREATEST(COALESCE("download", 0), 0),
    "upload" = GREATEST(COALESCE("upload", 0), 0)
WHERE "traffic" IS NULL OR "traffic" < 0
   OR "download" IS NULL OR "download" < 0
   OR "upload" IS NULL OR "upload" < 0;

ALTER TABLE "user_subscribe"
    ALTER COLUMN "traffic" SET NOT NULL,
    ALTER COLUMN "download" SET NOT NULL,
    ALTER COLUMN "upload" SET NOT NULL,
    ADD COLUMN IF NOT EXISTS "traffic_reset_at" timestamptz NOT NULL DEFAULT to_timestamp(0),
    ADD COLUMN IF NOT EXISTS "used_traffic" numeric(20, 0)
        GENERATED ALWAYS AS ("download"::numeric + "upload"::numeric) STORED;

ALTER TABLE "user_subscribe"
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_traffic_nonnegative",
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_download_nonnegative",
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_upload_nonnegative";

ALTER TABLE "user_subscribe"
    ADD CONSTRAINT "chk_user_subscribe_traffic_nonnegative" CHECK ("traffic" >= 0),
    ADD CONSTRAINT "chk_user_subscribe_download_nonnegative" CHECK ("download" >= 0),
    ADD CONSTRAINT "chk_user_subscribe_upload_nonnegative" CHECK ("upload" >= 0);

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_active_expiry_due"
    ON "user_subscribe" ("expire_time", "id")
    WHERE "status" = 1 AND "expire_time" <> to_timestamp(0);

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_active_traffic_due"
    ON "user_subscribe" ("id")
    WHERE "status" = 1 AND "traffic" > 0 AND "used_traffic" >= "traffic";

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_traffic_reset_at_id"
    ON "user_subscribe" ("traffic_reset_at", "id");
