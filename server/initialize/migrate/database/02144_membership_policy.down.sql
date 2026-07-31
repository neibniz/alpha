ALTER TABLE "membership_plan"
    DROP COLUMN IF EXISTS "monthly_order_limit",
    DROP COLUMN IF EXISTS "monthly_order_limit_enabled",
    DROP COLUMN IF EXISTS "monthly_consumption_limit",
    DROP COLUMN IF EXISTS "monthly_consumption_limit_enabled";
