CREATE INDEX IF NOT EXISTS "idx_traffic_log_server_timestamp_id_desc"
    ON "traffic_log" ("server_id", "timestamp" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_traffic_log_user_timestamp_id_desc"
    ON "traffic_log" ("user_id", "timestamp" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_traffic_log_timestamp_id_desc"
    ON "traffic_log" ("timestamp" DESC, "id" DESC);
