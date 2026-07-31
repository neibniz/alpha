DROP INDEX IF EXISTS "idx_user_subscribe_traffic_reset_due";
DROP INDEX IF EXISTS "idx_user_subscribe_expiry_due";

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_active_expiry_due"
    ON "user_subscribe" ("expire_time", "id")
    WHERE "status" = 1 AND "expire_time" <> to_timestamp(0);

ALTER TABLE "user_subscribe"
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_reset_cycle",
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_status",
    DROP COLUMN IF EXISTS "next_traffic_reset_at",
    DROP COLUMN IF EXISTS "reset_cycle",
    ALTER COLUMN "expire_time" DROP NOT NULL,
    ALTER COLUMN "expire_time" SET DEFAULT NULL;
