DROP INDEX IF EXISTS "idx_server_xray_templates_alias";

ALTER TABLE "server_xray_templates"
    DROP COLUMN IF EXISTS "subscription_variables",
    DROP COLUMN IF EXISTS "subscription_name",
    DROP COLUMN IF EXISTS "subscription_enabled",
    DROP COLUMN IF EXISTS "variables",
    DROP COLUMN IF EXISTS "alias";

ALTER TABLE "xray_templates"
    DROP COLUMN IF EXISTS "subscription_meta",
    DROP COLUMN IF EXISTS "default_variables",
    DROP COLUMN IF EXISTS "variables_schema",
    DROP COLUMN IF EXISTS "config_template";
