ALTER TABLE "nodes"
    ADD COLUMN IF NOT EXISTS "inbound_alias" VARCHAR(100) NOT NULL DEFAULT '';

UPDATE "nodes"
SET "inbound_alias" = "protocol"
WHERE "inbound_alias" = '' AND "protocol" <> '';

CREATE INDEX IF NOT EXISTS "idx_nodes_inbound_alias" ON "nodes" ("server_id", "inbound_alias");
