DROP INDEX IF EXISTS "idx_user_subscribe_traffic_reset_at_id";
DROP INDEX IF EXISTS "idx_user_subscribe_active_traffic_due";
DROP INDEX IF EXISTS "idx_user_subscribe_active_expiry_due";

ALTER TABLE "user_subscribe"
	DROP CONSTRAINT IF EXISTS "chk_user_subscribe_upload_nonnegative",
	DROP CONSTRAINT IF EXISTS "chk_user_subscribe_download_nonnegative",
	DROP CONSTRAINT IF EXISTS "chk_user_subscribe_traffic_nonnegative",
	DROP COLUMN IF EXISTS "used_traffic",
	DROP COLUMN IF EXISTS "traffic_reset_at";

ALTER TABLE "user_subscribe"
	ALTER COLUMN "traffic" DROP NOT NULL,
	ALTER COLUMN "download" DROP NOT NULL,
	ALTER COLUMN "upload" DROP NOT NULL;

ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_source",
    DROP COLUMN IF EXISTS "source";
