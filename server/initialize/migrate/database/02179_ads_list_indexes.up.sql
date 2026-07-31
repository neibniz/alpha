CREATE INDEX IF NOT EXISTS "idx_ads_status_id_desc"
    ON "ads" ("status", "id" DESC);
