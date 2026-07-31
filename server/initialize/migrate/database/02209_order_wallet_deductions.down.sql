ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_commission_deduction_nonnegative",
    DROP COLUMN IF EXISTS "commission_deduction";
