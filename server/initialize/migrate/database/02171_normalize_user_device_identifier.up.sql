DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'Identifier'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'identifier'
    ) THEN
        ALTER TABLE "user_device" RENAME COLUMN "Identifier" TO "identifier";
    ELSIF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'Identifier'
    ) AND EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'identifier'
    ) THEN
        UPDATE "user_device"
        SET "identifier" = COALESCE(NULLIF("identifier", ''), "Identifier")
        WHERE "Identifier" IS NOT NULL
          AND ("identifier" IS NULL OR "identifier" = '');

        ALTER TABLE "user_device" DROP COLUMN "Identifier";
    ELSIF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'identifier'
    ) THEN
        ALTER TABLE "user_device" ADD COLUMN "identifier" varchar(255) DEFAULT '';
    END IF;
END $$;

ALTER TABLE "user_device" ALTER COLUMN "identifier" TYPE varchar(255);
ALTER TABLE "user_device" ALTER COLUMN "identifier" SET DEFAULT '';

CREATE INDEX IF NOT EXISTS "idx_user_device_identifier"
    ON "user_device" ("identifier");
