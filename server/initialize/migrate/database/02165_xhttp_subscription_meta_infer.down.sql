BEGIN;

UPDATE "xray_templates"
SET
    "subscription_meta" = jsonb_set(
        jsonb_set(
            jsonb_set(
                COALESCE("subscription_meta", '{}'::jsonb),
                '{transport}',
                '"xhttp"'::jsonb,
                TRUE
            ),
            '{xhttp_mode}',
            '"stream-one"'::jsonb,
            TRUE
        ),
        '{xhttp_extra}',
        '{"xmux":{"maxConcurrency":0,"maxConnections":1,"cMaxReuseTimes":0,"hMaxRequestTimes":0,"hMaxReusableSecs":0,"hKeepAlivePeriod":0}}'::jsonb,
        TRUE
    ),
    "updated_at" = NOW()
WHERE
    "type" = 'inbound'
    AND "name" = 'VLESS Reality XHTTP';

COMMIT;
