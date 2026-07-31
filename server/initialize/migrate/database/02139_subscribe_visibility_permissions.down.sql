ALTER TABLE "subscribe"
    DROP COLUMN IF EXISTS "allow_reset_traffic",
    DROP COLUMN IF EXISTS "allow_renewal",
    DROP COLUMN IF EXISTS "user_visible";
