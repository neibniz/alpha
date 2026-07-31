BEGIN;

INSERT INTO "xray_templates" (
    "name",
    "type",
    "config",
    "config_template",
    "variables_schema",
    "default_variables",
    "subscription_meta",
    "description",
    "enabled",
    "created_at",
    "updated_at"
)
SELECT
    'Geodata',
    'geodata',
    $json${
      "cron": "0 4 * * *",
      "assets": [
        {
          "url": "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",
          "file": "geoip.dat"
        },
        {
          "url": "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat",
          "file": "geosite.dat"
        }
      ]
    }$json$::jsonb,
    $template${
      "cron": "{{ .Vars.cron }}",
      {{- if .Vars.outbound }}
      "outbound": "{{ .Vars.outbound }}",
      {{- end }}
      "assets": [
        {
          "url": "{{ .Vars.geoip_url }}",
          "file": "geoip.dat"
        },
        {
          "url": "{{ .Vars.geosite_url }}",
          "file": "geosite.dat"
        }
      ]
    }$template$,
    $json${
      "properties": {
        "cron": {
          "type": "string",
          "description": "Five-field cron expression evaluated in the Xray runtime local timezone."
        },
        "outbound": {
          "type": "string",
          "description": "Optional outbound tag used to download geodata assets."
        },
        "geoip_url": {
          "type": "string",
          "description": "HTTPS URL for geoip.dat."
        },
        "geosite_url": {
          "type": "string",
          "description": "HTTPS URL for geosite.dat-compatible data."
        }
      },
      "required": ["cron", "geoip_url", "geosite_url"]
    }$json$::jsonb,
    $json${
      "cron": "0 4 * * *",
      "outbound": "",
      "geoip_url": "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",
      "geosite_url": "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat"
    }$json$::jsonb,
    $json${}$json$::jsonb,
    'Default Xray geodata auto update template. Xray writes into its asset directory, so geoip.dat and geosite.dat must already exist in the image or host asset path.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'geodata' AND "name" = 'Geodata'
);

COMMIT;
