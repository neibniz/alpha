CREATE INDEX IF NOT EXISTS "idx_order_status_id_desc"
    ON "order" ("status", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_order_user_id_id_desc"
    ON "order" ("user_id", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_order_user_status_id_desc"
    ON "order" ("user_id", "status", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_order_subscribe_type_id_desc"
    ON "order" ("subscribe_id", "type", "id" DESC);
