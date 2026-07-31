CREATE INDEX IF NOT EXISTS "idx_order_user_created_type_status"
    ON "order" ("user_id", "created_at" DESC, "type", "status");
