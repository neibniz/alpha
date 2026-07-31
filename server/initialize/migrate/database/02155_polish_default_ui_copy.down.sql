UPDATE "system"
SET "value" = 'Alpha is a pure, professional, and perfect open-source proxy panel tool, designed to be your ideal choice for learning and practical use.',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "category" = 'site'
  AND "key" = 'SiteDesc'
  AND "value" = '专业、纯净的代理面板工具，适合学习与实践。';

UPDATE "subscribe_application"
SET "description" = CASE "description"
    WHEN '通用 URI 订阅模板。' THEN 'Default URI subscription template for generic clients.'
    WHEN 'Shadowrocket URI 订阅模板。' THEN 'Shadowrocket URI subscription template.'
    WHEN 'Clash/Mihomo YAML 订阅模板。' THEN 'Clash and Mihomo YAML subscription template.'
    WHEN 'sing-box 1.12 JSON 订阅模板。' THEN 'sing-box 1.12 JSON subscription template.'
    WHEN 'Surge 托管配置模板。' THEN 'Surge managed configuration template.'
    WHEN 'Stash YAML 订阅模板。' THEN 'Stash YAML subscription template.'
    WHEN 'Surfboard 配置模板。' THEN 'Surfboard configuration template.'
    WHEN 'Loon 配置模板。' THEN 'Loon configuration template.'
    WHEN 'Quantumult 配置模板。' THEN 'Quantumult configuration template.'
    WHEN 'Quantumult X 配置模板。' THEN 'Quantumult X configuration template.'
    WHEN 'Egern YAML 订阅模板。' THEN 'Egern YAML subscription template.'
    WHEN 'Hiddify JSON 订阅模板。' THEN 'Hiddify JSON subscription template.'
    WHEN 'sing-box 1.11 JSON 订阅模板。' THEN 'sing-box 1.11 JSON subscription template.'
    WHEN 'Xray-core 客户端 JSON 模板，为每个节点生成本地 SOCKS 入站与一对一路由。' THEN 'Xray-core client config with one local SOCKS inbound per node and one-to-one routing.'
    WHEN 'Surge 配置模板，连接本地 xray-core SOCKS 入站并复用 Surge 规则与策略组。' THEN 'Surge config that connects to local xray-core SOCKS inbounds while reusing the built-in Surge rules and policy groups.'
    ELSE "description"
END,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "description" IN (
    '通用 URI 订阅模板。',
    'Shadowrocket URI 订阅模板。',
    'Clash/Mihomo YAML 订阅模板。',
    'sing-box 1.12 JSON 订阅模板。',
    'Surge 托管配置模板。',
    'Stash YAML 订阅模板。',
    'Surfboard 配置模板。',
    'Loon 配置模板。',
    'Quantumult 配置模板。',
    'Quantumult X 配置模板。',
    'Egern YAML 订阅模板。',
    'Hiddify JSON 订阅模板。',
    'sing-box 1.11 JSON 订阅模板。',
    'Xray-core 客户端 JSON 模板，为每个节点生成本地 SOCKS 入站与一对一路由。',
    'Surge 配置模板，连接本地 xray-core SOCKS 入站并复用 Surge 规则与策略组。'
);

UPDATE "xray_templates"
SET "description" = CASE "description"
    WHEN 'VLESS Reality 入站模板，生产环境请替换密钥和伪装目标。' THEN 'Default VLESS Reality inbound. Replace the Reality key pair and camouflage target before production.'
    WHEN 'VLESS Reality gRPC 入站模板，生产环境请替换密钥和伪装目标。' THEN 'Default VLESS Reality gRPC inbound. Replace the Reality key pair and camouflage target before production.'
    WHEN 'DNS 模板。' THEN 'Default DNS template.'
    WHEN '阻断出站模板。' THEN 'Default blackhole outbound.'
    WHEN '直连出站模板。' THEN 'Default freedom outbound.'
    WHEN '安全路由模板。' THEN 'Default safety routing template.'
    WHEN '路由模板。' THEN 'Default routing template.'
    WHEN 'Xray 地理数据模板，用于声明 geoip.dat 与 geosite.dat 的自动更新信息。' THEN 'Default Xray geodata auto update template. Xray writes into its asset directory, so geoip.dat and geosite.dat must already exist in the image or host asset path.'
    ELSE "description"
END,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "description" IN (
    'VLESS Reality 入站模板，生产环境请替换密钥和伪装目标。',
    'VLESS Reality gRPC 入站模板，生产环境请替换密钥和伪装目标。',
    'DNS 模板。',
    '阻断出站模板。',
    '直连出站模板。',
    '安全路由模板。',
    '路由模板。',
    'Xray 地理数据模板，用于声明 geoip.dat 与 geosite.dat 的自动更新信息。'
);
