ALTER TABLE "membership_plan"
    ADD COLUMN IF NOT EXISTS "monthly_consumption_limit_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "monthly_consumption_limit" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "monthly_order_limit_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "monthly_order_limit" BIGINT NOT NULL DEFAULT 0;
