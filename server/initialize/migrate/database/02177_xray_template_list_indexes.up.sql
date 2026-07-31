CREATE INDEX IF NOT EXISTS "idx_xray_templates_type_enabled_id_desc"
    ON "xray_templates" ("type", "enabled", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_xray_templates_enabled_id_desc"
    ON "xray_templates" ("enabled", "id" DESC);
