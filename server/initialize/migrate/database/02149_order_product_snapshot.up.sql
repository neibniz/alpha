ALTER TABLE "order" ADD COLUMN IF NOT EXISTS "product_name" varchar(255) NOT NULL DEFAULT '';
ALTER TABLE "order" ADD COLUMN IF NOT EXISTS "product_snapshot" text NOT NULL DEFAULT '{}';

UPDATE "order" AS o
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
WHERE o."subscribe_id" = s."id"
  AND o."type" IN (1, 2, 3)
  AND o."product_snapshot" = '{}';

UPDATE "order" AS o
SET
    "product_name" = p."name",
    "product_snapshot" = json_build_object(
        'version', 1,
        'kind', 'membership',
        'id', p."id",
        'name', p."name",
        'description', p."description",
        'unit_price', p."unit_price",
        'duration_unit', p."duration_unit",
        'duration_value', p."duration_value",
        'membership_policy', json_build_object(
            'monthly_consumption_limit_enabled', p."monthly_consumption_limit_enabled",
            'monthly_consumption_limit', p."monthly_consumption_limit",
            'monthly_order_limit_enabled', p."monthly_order_limit_enabled",
            'monthly_order_limit', p."monthly_order_limit"
        )
    )::text
FROM "membership_plan" AS p
WHERE o."subscribe_id" = p."id"
  AND o."type" = 5
  AND o."product_snapshot" = '{}';

UPDATE "order"
SET
    "product_name" = 'Balance Recharge',
    "product_snapshot" = json_build_object(
        'version', 1,
        'kind', 'recharge',
        'name', 'Balance Recharge',
        'unit_price', "price"
    )::text
WHERE "type" = 4
  AND "product_snapshot" = '{}';
