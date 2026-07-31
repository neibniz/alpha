ALTER TABLE "user_subscribe" ADD COLUMN IF NOT EXISTS "product_name" varchar(255) NOT NULL DEFAULT '';
ALTER TABLE "user_subscribe" ADD COLUMN IF NOT EXISTS "product_snapshot" text NOT NULL DEFAULT '{}';

UPDATE "user_subscribe" AS us
SET
    "product_name" = o."product_name",
    "product_snapshot" = o."product_snapshot"
FROM "order" AS o
WHERE us."order_id" = o."id"
  AND o."product_snapshot" <> '{}'
  AND us."product_snapshot" = '{}';

UPDATE "user_subscribe" AS us
SET
    "product_name" = s."name",
    "product_snapshot" = json_build_object(
        'version', 1,
        'kind', 'subscribe',
        'id', s."id",
        'name', s."name",
        'description', s."description",
        'unit_price', s."unit_price",
        'unit_time', s."unit_time",
        'discount', CASE WHEN COALESCE(s."discount", '') = '' THEN '[]'::json ELSE s."discount"::json END,
        'replacement', s."replacement",
        'traffic', s."traffic",
        'speed_limit', s."speed_limit",
        'device_limit', s."device_limit",
        'quota', s."quota",
        'nodes', COALESCE((
            SELECT json_agg(TRIM(value)::bigint)
            FROM unnest(string_to_array(COALESCE(s."nodes", ''), ',')) AS value
            WHERE TRIM(value) <> ''
        ), '[]'::json),
        'node_tags', COALESCE((
            SELECT json_agg(TRIM(value))
            FROM unnest(string_to_array(COALESCE(s."node_tags", ''), ',')) AS value
            WHERE TRIM(value) <> ''
        ), '[]'::json),
        'show_original_price', s."show_original_price",
        'reset_cycle', s."reset_cycle",
        'renewal_reset', s."renewal_reset",
        'allow_renewal', s."allow_renewal",
        'allow_reset_traffic', s."allow_reset_traffic"
    )::text
FROM "subscribe" AS s
WHERE us."subscribe_id" = s."id"
  AND us."product_snapshot" = '{}';
