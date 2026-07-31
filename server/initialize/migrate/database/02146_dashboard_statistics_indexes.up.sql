CREATE INDEX IF NOT EXISTS "idx_order_dashboard_stats"
    ON "order" ("created_at", "status", "type");

CREATE INDEX IF NOT EXISTS "idx_order_dashboard_recent"
    ON "order" ("id", "type", "status");

CREATE INDEX IF NOT EXISTS "idx_user_dashboard_created_at"
    ON "user" ("created_at");

CREATE INDEX IF NOT EXISTS "idx_ticket_dashboard_status"
    ON "ticket" ("status");

CREATE INDEX IF NOT EXISTS "idx_system_logs_activity_recent"
    ON "system_logs" ("type", "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_subscribe_access_recent"
    ON "subscribe_access_logs" ("created_at", "id");
