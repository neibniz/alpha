DROP INDEX IF EXISTS "idx_nodes_inbound_alias";

ALTER TABLE "nodes"
    DROP COLUMN IF EXISTS "inbound_alias";
