UPDATE "xray_templates"
SET
    "config" = $json${
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
    }$json$::jsonb,
    "description" = 'Default safety routing template.',
    "updated_at" = NOW()
WHERE
    "type" = 'routing'
    AND "name" = 'Routing'
    AND "config" = $json${
      "domainStrategy": "AsIs",
      "rules": [
        {
          "type": "field",
          "protocol": ["bittorrent"],
          "outboundTag": "block"
        }
      ]
    }$json$::jsonb;
