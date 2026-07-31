DROP INDEX IF EXISTS "idx_order_user_type_request_id";

ALTER TABLE "order"
    DROP COLUMN IF EXISTS "request_id";
