ALTER TABLE "membership_plan"
    DROP COLUMN IF EXISTS "subscription_expiry_limit_enabled",
    DROP COLUMN IF EXISTS "subscription_expiry_grace_days";
