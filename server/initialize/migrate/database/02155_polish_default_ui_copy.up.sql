UPDATE "system"
SET "value" = '专业、纯净的代理面板工具，适合学习与实践。',
    "updated_at" = CURRENT_TIMESTAMP
WHERE "category" = 'site'
  AND "key" = 'SiteDesc'
  AND "value" = 'Alpha is a pure, professional, and perfect open-source proxy panel tool, designed to be your ideal choice for learning and practical use.';

UPDATE "subscribe_application"
SET "description" = CASE "description"
    WHEN 'Default URI subscription template for generic clients.' THEN '通用 URI 订阅模板。'
    WHEN 'Shadowrocket URI subscription template.' THEN 'Shadowrocket URI 订阅模板。'
    WHEN 'Clash and Mihomo YAML subscription template.' THEN 'Clash/Mihomo YAML 订阅模板。'
    WHEN 'sing-box 1.12 JSON subscription template.' THEN 'sing-box 1.12 JSON 订阅模板。'
    WHEN 'Surge managed configuration template.' THEN 'Surge 托管配置模板。'
    WHEN 'Stash YAML subscription template.' THEN 'Stash YAML 订阅模板。'
    WHEN 'Surfboard configuration template.' THEN 'Surfboard 配置模板。'
    WHEN 'Loon configuration template.' THEN 'Loon 配置模板。'
    WHEN 'Quantumult configuration template.' THEN 'Quantumult 配置模板。'
    WHEN 'Quantumult X configuration template.' THEN 'Quantumult X 配置模板。'
    WHEN 'Egern YAML subscription template.' THEN 'Egern YAML 订阅模板。'
    WHEN 'Hiddify JSON subscription template.' THEN 'Hiddify JSON 订阅模板。'
    WHEN 'sing-box 1.11 JSON subscription template.' THEN 'sing-box 1.11 JSON 订阅模板。'
    WHEN 'Xray-core client config with one local SOCKS inbound per node and one-to-one routing.' THEN 'Xray-core 客户端 JSON 模板，为每个节点生成本地 SOCKS 入站与一对一路由。'
    WHEN 'Surge config that connects to local xray-core SOCKS inbounds while reusing the built-in Surge rules and policy groups.' THEN 'Surge 配置模板，连接本地 xray-core SOCKS 入站并复用 Surge 规则与策略组。'
    ELSE "description"
END,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "description" IN (
    'Default URI subscription template for generic clients.',
    'Shadowrocket URI subscription template.',
    'Clash and Mihomo YAML subscription template.',
    'sing-box 1.12 JSON subscription template.',
    'Surge managed configuration template.',
    'Stash YAML subscription template.',
    'Surfboard configuration template.',
    'Loon configuration template.',
    'Quantumult configuration template.',
    'Quantumult X configuration template.',
    'Egern YAML subscription template.',
    'Hiddify JSON subscription template.',
    'sing-box 1.11 JSON subscription template.',
    'Xray-core client config with one local SOCKS inbound per node and one-to-one routing.',
    'Surge config that connects to local xray-core SOCKS inbounds while reusing the built-in Surge rules and policy groups.'
);

UPDATE "xray_templates"
SET "description" = CASE "description"
    WHEN 'Default VLESS Reality inbound. Replace the Reality key pair and camouflage target before production.' THEN 'VLESS Reality 入站模板，生产环境请替换密钥和伪装目标。'
    WHEN 'Default VLESS Reality gRPC inbound. Replace the Reality key pair and camouflage target before production.' THEN 'VLESS Reality gRPC 入站模板，生产环境请替换密钥和伪装目标。'
    WHEN 'Default DNS template.' THEN 'DNS 模板。'
    WHEN 'Default blackhole outbound.' THEN '阻断出站模板。'
    WHEN 'Default freedom outbound.' THEN '直连出站模板。'
    WHEN 'Default safety routing template.' THEN '安全路由模板。'
    WHEN 'Default routing template.' THEN '路由模板。'
    WHEN 'Default Xray geodata auto update template. Xray writes into its asset directory, so geoip.dat and geosite.dat must already exist in the image or host asset path.' THEN 'Xray 地理数据模板，用于声明 geoip.dat 与 geosite.dat 的自动更新信息。'
    ELSE "description"
END,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "description" IN (
    'Default VLESS Reality inbound. Replace the Reality key pair and camouflage target before production.',
    'Default VLESS Reality gRPC inbound. Replace the Reality key pair and camouflage target before production.',
    'Default DNS template.',
    'Default blackhole outbound.',
    'Default freedom outbound.',
    'Default safety routing template.',
    'Default routing template.',
    'Default Xray geodata auto update template. Xray writes into its asset directory, so geoip.dat and geosite.dat must already exist in the image or host asset path.'
);
