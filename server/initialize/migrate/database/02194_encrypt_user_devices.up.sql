ALTER TABLE "user_device"
    ALTER COLUMN "ip" TYPE text,
    ALTER COLUMN "identifier" TYPE text,
    ALTER COLUMN "user_agent" TYPE text,
    ADD COLUMN IF NOT EXISTS "identifier_bidx" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "identifier_display" varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "ip_display" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "user_agent_display" varchar(255) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS "idx_user_device_identifier";

CREATE INDEX IF NOT EXISTS "idx_user_device_identifier_bidx"
    ON "user_device" ("identifier_bidx")
    WHERE "identifier_bidx" <> '';
