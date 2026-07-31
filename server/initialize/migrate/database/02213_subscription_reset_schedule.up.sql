UPDATE "user_subscribe"
SET "expire_time" = to_timestamp(0)
WHERE "expire_time" IS NULL;

ALTER TABLE "user_subscribe"
    ALTER COLUMN "expire_time" SET DEFAULT to_timestamp(0),
    ALTER COLUMN "expire_time" SET NOT NULL,
    ADD COLUMN IF NOT EXISTS "reset_cycle" smallint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "next_traffic_reset_at" timestamptz NULL;

UPDATE "user_subscribe"
SET "reset_cycle" = CASE
    WHEN "product_snapshot" IS JSON
        AND COALESCE("product_snapshot"::jsonb ->> 'reset_cycle', '') ~ '^[0-3]$'
    THEN ("product_snapshot"::jsonb ->> 'reset_cycle')::smallint
    ELSE 0
END;

CREATE OR REPLACE FUNCTION pg_temp.alpha_next_traffic_reset_at(
    p_anchor timestamptz,
    p_cycle smallint,
    p_after timestamptz,
    p_expire timestamptz
) RETURNS timestamptz
LANGUAGE plpgsql
AS $$
DECLARE
    anchor_local timestamp;
    after_local timestamp;
    month_start timestamp;
    candidate_local timestamp;
    candidate_at timestamptz;
    anchor_day integer;
    last_day integer;
BEGIN
    IF p_cycle NOT BETWEEN 1 AND 3 THEN
        RETURN NULL;
    END IF;

    anchor_local := COALESCE(p_anchor, p_after) AT TIME ZONE 'Asia/Shanghai';
    after_local := p_after AT TIME ZONE 'Asia/Shanghai';
    anchor_day := EXTRACT(DAY FROM anchor_local)::integer;

    IF p_cycle = 1 THEN
        candidate_local := date_trunc('month', after_local) + interval '1 month';
    ELSIF p_cycle = 2 THEN
        month_start := date_trunc('month', after_local);
        last_day := EXTRACT(DAY FROM month_start + interval '1 month - 1 day')::integer;
        candidate_local := month_start + (LEAST(anchor_day, last_day) - 1) * interval '1 day';
        IF candidate_local <= after_local THEN
            month_start := month_start + interval '1 month';
            last_day := EXTRACT(DAY FROM month_start + interval '1 month - 1 day')::integer;
            candidate_local := month_start + (LEAST(anchor_day, last_day) - 1) * interval '1 day';
        END IF;
    ELSE
        candidate_local := make_date(
            EXTRACT(YEAR FROM after_local)::integer,
            EXTRACT(MONTH FROM anchor_local)::integer,
            1
        )::timestamp;
        last_day := EXTRACT(DAY FROM date_trunc('month', candidate_local) + interval '1 month - 1 day')::integer;
        candidate_local := candidate_local + (LEAST(anchor_day, last_day) - 1) * interval '1 day';
        IF candidate_local <= after_local THEN
            candidate_local := make_date(
                EXTRACT(YEAR FROM after_local)::integer + 1,
                EXTRACT(MONTH FROM anchor_local)::integer,
                1
            )::timestamp;
            last_day := EXTRACT(DAY FROM date_trunc('month', candidate_local) + interval '1 month - 1 day')::integer;
            candidate_local := candidate_local + (LEAST(anchor_day, last_day) - 1) * interval '1 day';
        END IF;
    END IF;

    candidate_at := candidate_local AT TIME ZONE 'Asia/Shanghai';
    IF p_expire <> to_timestamp(0) AND candidate_at >= p_expire THEN
        RETURN NULL;
    END IF;
    RETURN candidate_at;
END;
$$;

UPDATE "user_subscribe"
SET "next_traffic_reset_at" = pg_temp.alpha_next_traffic_reset_at(
    "start_time",
    "reset_cycle",
    CURRENT_TIMESTAMP,
    "expire_time"
);

ALTER TABLE "user_subscribe"
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_status",
    DROP CONSTRAINT IF EXISTS "chk_user_subscribe_reset_cycle";

ALTER TABLE "user_subscribe"
    ADD CONSTRAINT "chk_user_subscribe_status" CHECK ("status" BETWEEN 0 AND 5),
    ADD CONSTRAINT "chk_user_subscribe_reset_cycle" CHECK ("reset_cycle" BETWEEN 0 AND 3);

CREATE INDEX IF NOT EXISTS "idx_user_subscribe_traffic_reset_due"
    ON "user_subscribe" ("next_traffic_reset_at", "id")
    WHERE "status" IN (1, 2) AND "next_traffic_reset_at" IS NOT NULL;

DROP INDEX IF EXISTS "idx_user_subscribe_active_expiry_due";

CREATE INDEX "idx_user_subscribe_expiry_due"
    ON "user_subscribe" ("expire_time", "id")
    WHERE "status" IN (1, 2) AND "expire_time" <> to_timestamp(0);
