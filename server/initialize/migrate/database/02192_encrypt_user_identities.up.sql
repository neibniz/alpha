ALTER TABLE "user_auth_methods"
    ALTER COLUMN "auth_identifier" TYPE text,
    ADD COLUMN IF NOT EXISTS "auth_identifier_bidx" varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "auth_identifier_display" varchar(255) NOT NULL DEFAULT '';

ALTER TABLE "user_auth_methods"
    DROP CONSTRAINT IF EXISTS "idx_idx_auth_identifier";

DROP INDEX IF EXISTS "idx_auth_identifier";
DROP INDEX IF EXISTS "idx_user_auth_identifier_trgm";
DROP INDEX IF EXISTS "idx_user_auth_methods_type_user_identifier";

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_auth_identifier_bidx"
    ON "user_auth_methods" ("auth_identifier_bidx")
    WHERE "auth_identifier_bidx" <> '';

CREATE INDEX IF NOT EXISTS "idx_user_auth_type_bidx"
    ON "user_auth_methods" ("auth_type", "auth_identifier_bidx");
