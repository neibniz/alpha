CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS "idx_ticket_active_id_desc"
    ON "ticket" ("id" DESC)
    WHERE "status" <> 4;

CREATE INDEX IF NOT EXISTS "idx_ticket_user_active_id_desc"
    ON "ticket" ("user_id", "id" DESC)
    WHERE "status" <> 4;

CREATE INDEX IF NOT EXISTS "idx_ticket_status_id_desc"
    ON "ticket" ("status", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_ticket_title_trgm"
    ON "ticket" USING GIN ("title" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_ticket_description_trgm"
    ON "ticket" USING GIN ("description" gin_trgm_ops);
