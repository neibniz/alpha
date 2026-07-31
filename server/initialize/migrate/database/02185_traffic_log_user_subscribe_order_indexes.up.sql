CREATE INDEX IF NOT EXISTS "idx_traffic_log_user_sub_timestamp_id"
    ON "traffic_log" ("user_id", "subscribe_id", "timestamp" DESC, "id" DESC);
