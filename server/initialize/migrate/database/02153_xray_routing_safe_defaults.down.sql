UPDATE "xray_templates"
SET
    "config" = $json${
      "domainStrategy": "AsIs",
      "rules": [
        {
          "type": "field",
          "protocol": ["bittorrent"],
          "outboundTag": "block"
        }
      ]
    }$json$::jsonb,
    "description" = 'Default routing template.',
    "updated_at" = NOW()
WHERE
    "type" = 'routing'
    AND "name" = 'Routing'
    AND "config" = $json${
      "domainStrategy": "IPIfNonMatch",
      "rules": [
        {
          "type": "field",
          "ip": ["geoip:private"],
          "outboundTag": "block"
        },
        {
          "type": "field",
          "protocol": ["bittorrent"],
          "outboundTag": "block"
        }
      ]
    }$json$::jsonb;
