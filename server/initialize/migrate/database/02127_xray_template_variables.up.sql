ALTER TABLE "xray_templates"
    ADD COLUMN IF NOT EXISTS "config_template" TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "variables_schema" JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "default_variables" JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "subscription_meta" JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE "server_xray_templates"
    ADD COLUMN IF NOT EXISTS "alias" VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "variables" JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "subscription_enabled" BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "subscription_name" VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "subscription_variables" JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS "idx_server_xray_templates_alias" ON "server_xray_templates" ("server_id", "alias");
