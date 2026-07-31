ALTER TABLE "membership_plan"
    ADD COLUMN IF NOT EXISTS "subscription_expiry_limit_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "subscription_expiry_grace_days" BIGINT NOT NULL DEFAULT 0;
