ALTER TABLE "order"
    ADD COLUMN IF NOT EXISTS "settlement_amount" bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "settlement_currency" varchar(8) NOT NULL DEFAULT '';

ALTER TABLE "order"
    DROP CONSTRAINT IF EXISTS "chk_order_settlement_amount_nonnegative",
    DROP CONSTRAINT IF EXISTS "chk_order_settlement_pair";

ALTER TABLE "order"
    ADD CONSTRAINT "chk_order_settlement_amount_nonnegative"
        CHECK ("settlement_amount" >= 0),
    ADD CONSTRAINT "chk_order_settlement_pair"
        CHECK (("settlement_amount" = 0 AND "settlement_currency" = '')
            OR ("settlement_amount" > 0 AND "settlement_currency" <> ''));
