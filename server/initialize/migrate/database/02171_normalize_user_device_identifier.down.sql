DROP INDEX IF EXISTS "idx_user_device_identifier";

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'identifier'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_device'
          AND column_name = 'Identifier'
    ) THEN
        ALTER TABLE "user_device" RENAME COLUMN "identifier" TO "Identifier";
    END IF;
END $$;
