DROP INDEX IF EXISTS "idx_user_device_identifier_bidx";

ALTER TABLE "user_device"
    DROP COLUMN IF EXISTS "user_agent_display",
    DROP COLUMN IF EXISTS "ip_display",
    DROP COLUMN IF EXISTS "identifier_display",
    DROP COLUMN IF EXISTS "identifier_bidx";

CREATE INDEX IF NOT EXISTS "idx_user_device_identifier"
    ON "user_device" ("identifier");
