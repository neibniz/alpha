DROP INDEX IF EXISTS "idx_user_auth_type_bidx";
DROP INDEX IF EXISTS "idx_user_auth_identifier_bidx";

ALTER TABLE "user_auth_methods"
    DROP COLUMN IF EXISTS "auth_identifier_display",
    DROP COLUMN IF EXISTS "auth_identifier_bidx";

ALTER TABLE "user_auth_methods"
    ADD CONSTRAINT "idx_idx_auth_identifier" UNIQUE ("auth_identifier");

CREATE INDEX IF NOT EXISTS "idx_user_auth_identifier_trgm"
    ON "user_auth_methods" USING GIN ("auth_identifier" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_user_auth_methods_type_user_identifier"
    ON "user_auth_methods" ("auth_type", "user_id", "auth_identifier");
