CREATE INDEX IF NOT EXISTS "idx_payment_enable_id_desc"
    ON "payment" ("enable", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_payment_platform_id_desc"
    ON "payment" ("platform", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_payment_platform_enable_id_desc"
    ON "payment" ("platform", "enable", "id" DESC);
