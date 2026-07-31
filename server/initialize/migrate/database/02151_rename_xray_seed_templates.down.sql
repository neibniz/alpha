BEGIN;

UPDATE "xray_templates"
SET
    "name" = CASE
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality' THEN 'VLESS Reality Inbound'
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality gRPC' THEN 'VLESS Reality gRPC Inbound'
        WHEN "type" = 'inbound' AND "name" = 'VLESS Reality XHTTP' THEN 'VLESS Reality XHTTP Inbound'
        WHEN "type" = 'inbound' AND "name" = 'Hysteria2' THEN 'Hysteria2 Inbound'
        WHEN "type" = 'outbound' AND "name" = 'Direct' THEN 'Direct Outbound'
        WHEN "type" = 'outbound' AND "name" = 'Block' THEN 'Block Outbound'
        WHEN "type" = 'dns' AND "name" = 'DNS' THEN 'Default DNS'
        WHEN "type" = 'routing' AND "name" = 'Routing' THEN 'Default Routing'
        WHEN "type" = 'geodata' AND "name" = 'Geodata' THEN 'Geodata Auto Update'
        ELSE "name"
    END,
    "updated_at" = NOW()
WHERE ("type", "name") IN (
    ('inbound', 'VLESS Reality'),
    ('inbound', 'VLESS Reality gRPC'),
    ('inbound', 'VLESS Reality XHTTP'),
    ('inbound', 'Hysteria2'),
    ('outbound', 'Direct'),
    ('outbound', 'Block'),
    ('dns', 'DNS'),
    ('routing', 'Routing'),
    ('geodata', 'Geodata')
);

COMMIT;
