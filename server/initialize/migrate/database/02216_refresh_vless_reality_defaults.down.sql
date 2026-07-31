BEGIN;

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,realitySettings,target}',
      to_jsonb('www.apple.com:443'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "config" #>> '{streamSettings,realitySettings,target}' = 'www.ebay.com:443';

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,realitySettings,serverNames}',
      '["www.apple.com"]'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "config" #> '{streamSettings,realitySettings,serverNames}' = '["www.ebay.com"]'::jsonb;

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{reality_target}',
      to_jsonb('www.apple.com:443'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "default_variables" ->> 'reality_target' = 'www.ebay.com:443';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{reality_server_names}',
      '["www.apple.com"]'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "default_variables" -> 'reality_server_names' = '["www.ebay.com"]'::jsonb;

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,grpcSettings,authority}',
      to_jsonb('www.apple.com'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "config" #>> '{streamSettings,grpcSettings,authority}' = 'www.ebay.com';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{grpc_authority}',
      to_jsonb('www.apple.com'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "default_variables" ->> 'grpc_authority' = 'www.ebay.com';

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{settings,users,0,flow}',
      '"xtls-rprx-vision"'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "config" #>> '{settings,users,0,flow}' = 'none';

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{settings,clients,0,flow}',
      '"xtls-rprx-vision"'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "config" #>> '{settings,clients,0,flow}' = 'none';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{flow}',
      '"xtls-rprx-vision"'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "default_variables" ->> 'flow' = 'none';

COMMIT;
