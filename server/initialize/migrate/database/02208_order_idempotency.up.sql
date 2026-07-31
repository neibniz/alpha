ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS "request_id" varchar(64) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS "idx_order_user_type_request_id"
    ON "order" ("user_id", "type", "request_id")
    WHERE "request_id" <> '';
