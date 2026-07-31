BEGIN;

-- Refresh only untouched built-in camouflage defaults. Custom targets are preserved.
UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,realitySettings,target}',
      to_jsonb('www.ebay.com:443'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "config" #>> '{streamSettings,realitySettings,target}' = 'www.apple.com:443';

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,realitySettings,serverNames}',
      '["www.ebay.com"]'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "config" #> '{streamSettings,realitySettings,serverNames}' = '["www.apple.com"]'::jsonb;

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{reality_target}',
      to_jsonb('www.ebay.com:443'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "default_variables" ->> 'reality_target' = 'www.apple.com:443';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{reality_server_names}',
      '["www.ebay.com"]'::jsonb,
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" IN ('VLESS Reality', 'VLESS Reality gRPC', 'VLESS Reality XHTTP')
  AND "default_variables" -> 'reality_server_names' = '["www.apple.com"]'::jsonb;

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{streamSettings,grpcSettings,authority}',
      to_jsonb('www.ebay.com'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "config" #>> '{streamSettings,grpcSettings,authority}' = 'www.apple.com';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{grpc_authority}',
      to_jsonb('www.ebay.com'::text),
      false
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND "default_variables" ->> 'grpc_authority' = 'www.apple.com';

-- "none" is the management-layer value. Runtime rendering omits it for xray-core.
UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{settings,users,0,flow}',
      '"none"'::jsonb,
      true
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND jsonb_typeof("config" #> '{settings,users,0}') = 'object';

UPDATE "xray_templates"
SET "config" = jsonb_set(
      "config",
      '{settings,clients,0,flow}',
      '"none"'::jsonb,
      true
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC'
  AND jsonb_typeof("config" #> '{settings,clients,0}') = 'object';

UPDATE "xray_templates"
SET "default_variables" = jsonb_set(
      "default_variables",
      '{flow}',
      '"none"'::jsonb,
      true
    ),
    "updated_at" = NOW()
WHERE "type" = 'inbound'
  AND "name" = 'VLESS Reality gRPC';

COMMIT;
