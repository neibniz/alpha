CREATE UNIQUE INDEX IF NOT EXISTS "idx_daily_user_traffic_unique"
    ON "daily_user_traffic" ("date", "user_id", "subscribe_id");

CREATE UNIQUE INDEX IF NOT EXISTS "idx_daily_server_traffic_unique"
    ON "daily_server_traffic" ("date", "server_id");

INSERT INTO "daily_user_traffic" ("date", "user_id", "subscribe_id", "download", "upload", "total", "created_at", "updated_at")
SELECT
    TO_CHAR("timestamp", 'YYYY-MM-DD') AS "date",
    "user_id",
    "subscribe_id",
    COALESCE(SUM("download"), 0) AS "download",
    COALESCE(SUM("upload"), 0) AS "upload",
    COALESCE(SUM("download" + "upload"), 0) AS "total",
    NOW(),
    NOW()
FROM "traffic_log"
GROUP BY TO_CHAR("timestamp", 'YYYY-MM-DD'), "user_id", "subscribe_id"
ON CONFLICT ("date", "user_id", "subscribe_id") DO UPDATE SET
    "download" = EXCLUDED."download",
    "upload" = EXCLUDED."upload",
    "total" = EXCLUDED."total",
    "updated_at" = NOW();

INSERT INTO "daily_server_traffic" ("date", "server_id", "download", "upload", "total", "created_at", "updated_at")
SELECT
    TO_CHAR("timestamp", 'YYYY-MM-DD') AS "date",
    "server_id",
    COALESCE(SUM("download"), 0) AS "download",
    COALESCE(SUM("upload"), 0) AS "upload",
    COALESCE(SUM("download" + "upload"), 0) AS "total",
    NOW(),
    NOW()
FROM "traffic_log"
GROUP BY TO_CHAR("timestamp", 'YYYY-MM-DD'), "server_id"
ON CONFLICT ("date", "server_id") DO UPDATE SET
    "download" = EXCLUDED."download",
    "upload" = EXCLUDED."upload",
    "total" = EXCLUDED."total",
    "updated_at" = NOW();

INSERT INTO "daily_traffic_stat" ("date", "download", "upload", "total", "created_at", "updated_at")
SELECT
    TO_CHAR("timestamp", 'YYYY-MM-DD') AS "date",
    COALESCE(SUM("download"), 0) AS "download",
    COALESCE(SUM("upload"), 0) AS "upload",
    COALESCE(SUM("download" + "upload"), 0) AS "total",
    NOW(),
    NOW()
FROM "traffic_log"
GROUP BY TO_CHAR("timestamp", 'YYYY-MM-DD')
ON CONFLICT ("date") DO UPDATE SET
    "download" = EXCLUDED."download",
    "upload" = EXCLUDED."upload",
    "total" = EXCLUDED."total",
    "updated_at" = NOW();
