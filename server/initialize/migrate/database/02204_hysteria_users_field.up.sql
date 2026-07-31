BEGIN;

UPDATE "xray_templates"
SET
    "config" = CASE
        WHEN "config" #> '{settings,users}' IS NULL THEN
            jsonb_set(
                "config" #- '{settings,clients}',
                '{settings,users}',
                "config" #> '{settings,clients}',
                TRUE
            )
        ELSE "config" #- '{settings,clients}'
    END,
    "description" = REPLACE("description", 'settings.clients', 'settings.users'),
    "updated_at" = NOW()
WHERE
    "type" = 'inbound'
    AND "config" ->> 'protocol' = 'hysteria'
    AND "config" #> '{settings,clients}' IS NOT NULL;

COMMIT;
