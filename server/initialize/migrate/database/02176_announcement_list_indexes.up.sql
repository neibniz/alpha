CREATE INDEX IF NOT EXISTS "idx_announcement_visibility_id_desc"
    ON "announcement" ("show", "pinned", "popup", "id" DESC);
