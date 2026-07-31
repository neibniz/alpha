ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_settlement_pair",
    DROP CONSTRAINT IF EXISTS "chk_order_settlement_amount_nonnegative",
    DROP COLUMN IF EXISTS "settlement_currency",
    DROP COLUMN IF EXISTS "settlement_amount";
