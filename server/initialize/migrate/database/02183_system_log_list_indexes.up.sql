CREATE INDEX IF NOT EXISTS "idx_system_logs_type_date_id"
    ON "system_logs" ("type", "date", "id" DESC);
