BEGIN;

UPDATE "xray_templates"
SET
    "name" = CASE
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality Inbound' THEN 'VLESS Reality'
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality gRPC Inbound' THEN 'VLESS Reality gRPC'
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality XHTTP Inbound' THEN 'VLESS Reality XHTTP'
        WHEN "type" = 'inbound' AND "name" = 'Hysteria2 Inbound' THEN 'Hysteria2'
        WHEN "type" = 'outbound' AND "name" = 'Direct Outbound' THEN 'Direct'
        WHEN "type" = 'outbound' AND "name" = 'Block Outbound' THEN 'Block'
        WHEN "type" = 'dns' AND "name" = 'Default DNS' THEN 'DNS'
        WHEN "type" = 'routing' AND "name" = 'Default Routing' THEN 'Routing'
        WHEN "type" = 'geodata' AND "name" = 'Geodata Auto Update' THEN 'Geodata'
        ELSE "name"
    END,
    "updated_at" = NOW()
WHERE ("type", "name") IN (
    ('inbound', 'VLESS Reality Inbound'),
    ('inbound', 'VLESS Reality gRPC Inbound'),
    ('inbound', 'VLESS Reality XHTTP Inbound'),
    ('inbound', 'Hysteria2 Inbound'),
    ('outbound', 'Direct Outbound'),
    ('outbound', 'Block Outbound'),
    ('dns', 'Default DNS'),
    ('routing', 'Default Routing'),
    ('geodata', 'Geodata Auto Update')
);

COMMIT;
