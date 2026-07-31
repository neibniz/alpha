CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS "idx_user_auth_identifier_trgm"
    ON "user_auth_methods" USING GIN ("auth_identifier" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_user_refer_code_trgm"
    ON "user" USING GIN ("refer_code" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_subscribe_status_user"
    ON "user_subscribe" ("subscribe_id", "status", "user_id");

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_user_status_id"
    ON "user_subscribe" ("user_id", "status", "id" DESC);
