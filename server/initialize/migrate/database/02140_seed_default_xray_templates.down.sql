DELETE FROM "xray_templates"
WHERE ("type", "name") IN (
    ('inbound', 'VLESS Reality'),
    ('inbound', 'VLESS Reality gRPC'),
    ('inbound', 'VLESS Reality XHTTP'),
    ('inbound', 'Hysteria2'),
    ('outbound', 'Direct'),
    ('outbound', 'Block'),
    ('dns', 'DNS'),
    ('routing', 'Routing')
);
