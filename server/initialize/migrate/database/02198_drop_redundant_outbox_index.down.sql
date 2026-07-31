CREATE INDEX IF NOT EXISTS "idx_traffic_event_outbox_pending"
    ON "traffic_event_outbox" ("id", "created_at");
