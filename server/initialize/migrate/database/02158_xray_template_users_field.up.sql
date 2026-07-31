BEGIN;

UPDATE "xray_templates"
SET
    "config" = jsonb_set(
        "config" #- '{settings,clients}',
        '{settings,users}',
        "config" #> '{settings,clients}',
        TRUE
    ),
    "description" = REPLACE("description", 'settings.clients', 'settings.users'),
    "updated_at" = NOW()
WHERE
    "type" = 'inbound'
    AND "config" #> '{settings,clients}' IS NOT NULL
    AND "config" ->> 'protocol' IN ('vless', 'vmess', 'trojan', 'shadowsocks');

COMMIT;
