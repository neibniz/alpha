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
    'VLESS Reality',
    'inbound',
    $json${
      "tag": "vless-reality",
      "listen": "0.0.0.0",
      "port": 443,
      "protocol": "vless",
      "settings": {
        "decryption": "none",
        "users": [
          {
            "id": "00000000-0000-0000-0000-000000000000",
            "flow": "xtls-rprx-vision",
            "email": "template"
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "target": "www.ebay.com:443",
          "privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
          "serverNames": ["www.ebay.com"],
          "shortIds": ["0123456789abcdef"]
        }
      }
    }$json$::jsonb,
    '',
    $json${
      "properties": {
        "tag": { "type": "string" },
        "listen": { "type": "string" },
        "port": { "type": "number" },
        "flow": { "type": "string" },
        "reality_target": { "type": "string" },
        "reality_private_key": { "type": "string" },
        "reality_server_names": { "type": "array" },
        "reality_short_ids": { "type": "array" }
      },
      "required": ["port", "reality_target", "reality_private_key", "reality_server_names", "reality_short_ids"]
    }$json$::jsonb,
    $json${
      "tag": "vless-reality",
      "listen": "0.0.0.0",
      "port": 443,
      "flow": "xtls-rprx-vision",
      "transport": "tcp",
      "security": "reality",
      "reality_target": "www.ebay.com:443",
      "reality_private_key": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
      "reality_server_names": ["www.ebay.com"],
      "reality_short_ids": ["0123456789abcdef"]
    }$json$::jsonb,
    $json${}$json$::jsonb,
    'Default VLESS Reality inbound. Replace the Reality key pair and camouflage target before production.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'inbound' AND "name" = 'VLESS Reality'
);

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
    'VLESS Reality gRPC',
    'inbound',
    $json${
      "tag": "vless-reality-grpc",
      "listen": "0.0.0.0",
      "port": 443,
      "protocol": "vless",
      "settings": {
        "decryption": "none",
        "users": [
          {
            "id": "00000000-0000-0000-0000-000000000000",
            "flow": "none",
            "email": "template"
          }
        ]
      },
      "streamSettings": {
        "network": "grpc",
        "security": "reality",
        "realitySettings": {
          "target": "www.ebay.com:443",
          "privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
          "serverNames": ["www.ebay.com"],
          "shortIds": ["0123456789abcdef"]
        },
        "grpcSettings": {
          "serviceName": "grpc",
          "authority": "www.ebay.com"
        }
      }
    }$json$::jsonb,
    '',
    $json${
      "properties": {
        "tag": { "type": "string" },
        "listen": { "type": "string" },
        "port": { "type": "number" },
        "flow": { "type": "string" },
        "reality_target": { "type": "string" },
        "reality_private_key": { "type": "string" },
        "reality_server_names": { "type": "array" },
        "reality_short_ids": { "type": "array" },
        "grpc_service_name": { "type": "string" },
        "grpc_authority": { "type": "string" }
      },
      "required": ["port", "reality_target", "reality_private_key", "reality_server_names", "reality_short_ids", "grpc_service_name"]
    }$json$::jsonb,
    $json${
      "tag": "vless-reality-grpc",
      "listen": "0.0.0.0",
      "port": 443,
      "flow": "none",
      "transport": "grpc",
      "security": "reality",
      "reality_target": "www.ebay.com:443",
      "reality_private_key": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
      "reality_server_names": ["www.ebay.com"],
      "reality_short_ids": ["0123456789abcdef"],
      "grpc_service_name": "grpc",
      "grpc_authority": "www.ebay.com"
    }$json$::jsonb,
    $json${}$json$::jsonb,
    'Default VLESS Reality gRPC inbound. Replace the Reality key pair and camouflage target before production.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'inbound' AND "name" = 'VLESS Reality gRPC'
);

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
    'VLESS Reality XHTTP',
    'inbound',
    $json${
      "tag": "vless-reality-xhttp",
      "listen": "0.0.0.0",
      "port": 443,
      "protocol": "vless",
      "settings": {
        "decryption": "none",
        "users": [
          {
            "id": "00000000-0000-0000-0000-000000000000",
            "email": "template"
          }
        ]
      },
      "streamSettings": {
        "network": "xhttp",
        "security": "reality",
        "realitySettings": {
          "target": "www.ebay.com:443",
          "privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
          "serverNames": ["www.ebay.com"],
          "shortIds": ["0123456789abcdef"]
        },
        "xhttpSettings": {
          "path": "/xhttp",
          "mode": "stream-one",
          "extra": {
            "xmux": {
              "maxConcurrency": 0,
              "maxConnections": 1,
              "cMaxReuseTimes": 0,
              "hMaxRequestTimes": 0,
              "hMaxReusableSecs": 0,
              "hKeepAlivePeriod": 0
            }
          }
        }
      }
    }$json$::jsonb,
    '',
    $json${
      "properties": {
        "tag": { "type": "string" },
        "listen": { "type": "string" },
        "port": { "type": "number" },
        "reality_target": { "type": "string" },
        "reality_private_key": { "type": "string" },
        "reality_server_names": { "type": "array" },
        "reality_short_ids": { "type": "array" },
        "path": { "type": "string" },
        "xhttp_mode": { "type": "string" },
        "xhttp_extra": { "type": "object" }
      },
      "required": ["port", "reality_target", "reality_private_key", "reality_server_names", "reality_short_ids", "path", "xhttp_mode", "xhttp_extra"]
    }$json$::jsonb,
    $json${
      "tag": "vless-reality-xhttp",
      "listen": "0.0.0.0",
      "port": 443,
      "transport": "xhttp",
      "security": "reality",
      "reality_target": "www.ebay.com:443",
      "reality_private_key": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
      "reality_server_names": ["www.ebay.com"],
      "reality_short_ids": ["0123456789abcdef"],
      "path": "/xhttp",
      "xhttp_mode": "stream-one",
      "xhttp_extra": {
        "xmux": {
          "maxConcurrency": 0,
          "maxConnections": 1,
          "cMaxReuseTimes": 0,
          "hMaxRequestTimes": 0,
          "hMaxReusableSecs": 0,
          "hKeepAlivePeriod": 0
        }
      }
    }$json$::jsonb,
    $json${
      "transport": "xhttp",
      "xhttp_mode": "stream-one",
      "xhttp_extra": {
        "xmux": {
          "maxConcurrency": 0,
          "maxConnections": 1,
          "cMaxReuseTimes": 0,
          "hMaxRequestTimes": 0,
          "hMaxReusableSecs": 0,
          "hKeepAlivePeriod": 0
        }
      }
    }$json$::jsonb,
    'Default VLESS Reality XHTTP inbound. XHTTP extra sets XMUX maxConnections to 1 so clients reuse one TCP connection for transport.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'inbound' AND "name" = 'VLESS Reality XHTTP'
);

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
    'Hysteria2',
    'inbound',
    $json${
      "tag": "hysteria2",
      "listen": "0.0.0.0",
      "port": 443,
      "protocol": "hysteria",
      "settings": {
        "version": 2,
        "users": [
          {
            "auth": "template-password",
            "level": 0,
            "email": "template"
          }
        ]
      },
      "streamSettings": {
        "network": "hysteria",
        "hysteriaSettings": {
          "version": 2,
          "udpIdleTimeout": 60,
          "masquerade": {
            "type": "string",
            "content": "ok",
            "headers": {
              "content-type": "text/plain"
            },
            "statusCode": 200
          }
        }
      }
    }$json$::jsonb,
    '',
    $json${
      "properties": {
        "tag": { "type": "string" },
        "listen": { "type": "string" },
        "port": { "type": "number" },
        "hysteria_version": { "type": "number" },
        "udp_idle_timeout": { "type": "number" },
        "hysteria_masquerade": { "type": "object" }
      },
      "required": ["port", "hysteria_version"]
    }$json$::jsonb,
    $json${
      "tag": "hysteria2",
      "listen": "0.0.0.0",
      "port": 443,
      "transport": "hysteria",
      "hysteria_version": 2,
      "udp_idle_timeout": 60,
      "hysteria_masquerade": {
        "type": "string",
        "content": "ok",
        "headers": {
          "content-type": "text/plain"
        },
        "statusCode": 200
      }
    }$json$::jsonb,
    $json${
      "transport": "hysteria",
      "type": "hysteria2"
    }$json$::jsonb,
    'Default Hysteria2 inbound. xray-agent injects real users into settings.users with auth, level and email.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'inbound' AND "name" = 'Hysteria2'
);

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
    'Direct',
    'outbound',
    $json${"tag": "direct", "protocol": "freedom"}$json$::jsonb,
    '',
    $json${"properties": {"tag": {"type": "string"}}}$json$::jsonb,
    $json${"tag": "direct"}$json$::jsonb,
    $json${}$json$::jsonb,
    'Default freedom outbound.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'outbound' AND "name" = 'Direct'
);

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
    'Block',
    'outbound',
    $json${"tag": "block", "protocol": "blackhole"}$json$::jsonb,
    '',
    $json${"properties": {"tag": {"type": "string"}}}$json$::jsonb,
    $json${"tag": "block"}$json$::jsonb,
    $json${}$json$::jsonb,
    'Default blackhole outbound.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'outbound' AND "name" = 'Block'
);

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
    'DNS',
    'dns',
    $json${
      "servers": [
        "https://1.1.1.1/dns-query",
        "8.8.8.8",
        "1.1.1.1"
      ]
    }$json$::jsonb,
    '',
    $json${}$json$::jsonb,
    $json${}$json$::jsonb,
    $json${}$json$::jsonb,
    'DNS template.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'dns' AND "name" = 'DNS'
);

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
    'Routing',
    'routing',
    $json${
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
    '',
    $json${}$json$::jsonb,
    $json${}$json$::jsonb,
    $json${}$json$::jsonb,
    'Default safety routing template.',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "xray_templates" WHERE "type" = 'routing' AND "name" = 'Routing'
);

COMMIT;
