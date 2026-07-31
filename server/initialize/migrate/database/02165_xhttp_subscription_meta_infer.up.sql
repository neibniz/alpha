BEGIN;

UPDATE "xray_templates"
SET
    "subscription_meta" = COALESCE("subscription_meta", '{}'::jsonb)
        - 'transport'
        - 'xhttp_mode'
        - 'xhttp_extra',
    "updated_at" = NOW()
WHERE
    "type" = 'inbound'
    AND "name" = 'VLESS Reality XHTTP'
    AND COALESCE("subscription_meta", '{}'::jsonb) ->> 'xhttp_mode' = 'stream-one';

COMMIT;
