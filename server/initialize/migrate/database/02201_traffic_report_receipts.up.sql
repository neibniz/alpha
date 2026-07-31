CREATE TABLE IF NOT EXISTS "traffic_report_receipt" (
    "report_id" UUID NOT NULL,
    "server_id" BIGINT NOT NULL,
    "sampled_at" TIMESTAMPTZ NOT NULL,
    "interval_seconds" INTEGER NOT NULL,
    "sequence" BIGINT NOT NULL DEFAULT 0,
    "processed_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("report_id", "server_id")
);

CREATE INDEX IF NOT EXISTS "idx_traffic_report_receipt_processed_at"
    ON "traffic_report_receipt" ("processed_at");
