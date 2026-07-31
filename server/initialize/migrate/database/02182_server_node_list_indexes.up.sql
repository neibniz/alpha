CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS "idx_servers_sort_id"
    ON "servers" ("sort", "id");

CREATE INDEX IF NOT EXISTS "idx_servers_name_trgm"
    ON "servers" USING GIN ("name" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_servers_address_trgm"
    ON "servers" USING GIN ("address" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_nodes_sort_id"
    ON "nodes" ("sort", "id");

CREATE INDEX IF NOT EXISTS "idx_nodes_server_sort_id"
    ON "nodes" ("server_id", "sort", "id");

CREATE INDEX IF NOT EXISTS "idx_nodes_enabled_sort_id"
    ON "nodes" ("enabled", "sort", "id");

CREATE INDEX IF NOT EXISTS "idx_nodes_name_trgm"
    ON "nodes" USING GIN ("name" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_nodes_address_trgm"
    ON "nodes" USING GIN ("address" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_nodes_tags_trgm"
    ON "nodes" USING GIN ("tags" gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_nodes_port_text_trgm"
    ON "nodes" USING GIN ((("port")::text) gin_trgm_ops);
