INSERT INTO "system" ("category", "key", "value", "type", "desc", "created_at", "updated_at")
VALUES ('server', 'NodeSecret', '', 'string', 'legacy node secret (disabled)', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("key") DO NOTHING;
