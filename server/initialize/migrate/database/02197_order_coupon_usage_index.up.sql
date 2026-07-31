CREATE INDEX IF NOT EXISTS "idx_order_user_coupon_status"
    ON "order" ("user_id", "coupon", "status", "id")
    WHERE "coupon" IS NOT NULL AND "coupon" <> '';
