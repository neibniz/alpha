BEGIN;

UPDATE "xray_templates"
SET
    "config" = CASE
        WHEN "config" #> '{settings,clients}' IS NULL THEN
            jsonb_set(
                "config" #- '{settings,users}',
                '{settings,clients}',
                "config" #> '{settings,users}',
                TRUE
            )
        ELSE
            "config" #- '{settings,users}'
    END,
    "description" = REPLACE("description", 'settings.users', 'settings.clients'),
    "updated_at" = NOW()
WHERE
    "type" = 'inbound'
    AND "config" #> '{settings,users}' IS NOT NULL
    AND "config" ->> 'protocol' IN ('vless', 'vmess', 'trojan', 'shadowsocks', 'hysteria');

COMMIT;
