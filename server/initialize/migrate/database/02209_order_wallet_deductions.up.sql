ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS "commission_deduction" bigint NOT NULL DEFAULT 0;

ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_commission_deduction_nonnegative";

ALTER TABLE "order"
    ADD CONSTRAINT "chk_order_commission_deduction_nonnegative"
        CHECK ("commission_deduction" >= 0);
