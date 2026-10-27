export type XrayTemplateType =
  | "inbound"
  | "outbound"
  | "dns"
  | "routing"
  | "geodata";

export const XRAY_TEMPLATE_TYPES: XrayTemplateType[] = [
  "inbound",
  "outbound",
  "dns",
  "routing",
  "geodata",
];

export const INBOUND_PROTOCOLS = [
  "vless",
  "vmess",
  "trojan",
  "shadowsocks",
  "socks",
  "http",
  "tunnel",
  "wireguard",
  "hysteria",
  "tun",
] as const;

export const OUTBOUND_PROTOCOLS = [
  "freedom",
  "blackhole",
  "dns",
  "loopback",
  "socks",
  "http",
  "shadowsocks",
  "vless",
  "vmess",
  "trojan",
  "wireguard",
  "hysteria",
] as const;

export const NETWORKS = [
  "raw",
  "websocket",
  "grpc",
  "xhttp",
  "httpupgrade",
  "mkcp",
  "hysteria",
] as const;
export const SECURITIES = ["none", "tls", "reality"] as const;
export const QUERY_STRATEGIES = [
  "UseIP",
  "UseIPv4",
  "UseIPv6",
  "UseSystem",
] as const;
export const ROUTING_DOMAIN_STRATEGIES = [
  "AsIs",
  "IPIfNonMatch",
  "IPOnDemand",
] as const;
export const ROUTING_PRESETS = [
  {
    id: "safety",
    label: "基础安全",
    description: "阻断内网地址与 BT 流量，适合作为节点默认路由。",
    domainStrategy: "IPIfNonMatch",
    rules: [
      {
        type: "field",
        ip: ["geoip:private"],
        outboundTag: "block",
      },
      {
        type: "field",
        protocol: ["bittorrent"],
        outboundTag: "block",
      },
    ],
    balancers: [],
  },
  {
    id: "anti-abuse",
    label: "防滥用增强",
    description: "在基础安全上额外阻断常见邮件端口，降低滥用风险。",
    domainStrategy: "IPIfNonMatch",
    rules: [
      {
        type: "field",
        ip: ["geoip:private"],
        outboundTag: "block",
      },
      {
        type: "field",
        protocol: ["bittorrent"],
        outboundTag: "block",
      },
      {
        type: "field",
        port: "25,465,587",
        outboundTag: "block",
      },
    ],
    balancers: [],
  },
] as const;

type JsonObject = Record<string, unknown>;
type XrayFormValues = Record<string, unknown>;

const LEGACY_NETWORK_ALIASES: Record<string, string> = {
  tcp: "raw",
  ws: "websocket",
  kcp: "mkcp",
};

export function normalizeTransportNetwork(value: unknown) {
  const network = typeof value === "string" ? value.trim().toLowerCase() : "";
  return LEGACY_NETWORK_ALIASES[network] || network;
}

export function normalizeInboundProtocol(value: unknown) {
  const protocol = typeof value === "string" ? value.trim().toLowerCase() : "";
  return protocol === "dokodemo-door" ? "tunnel" : protocol;
}

function isJsonObject(value: unknown): value is JsonObject {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function objectValue(value: unknown): JsonObject {
  return isJsonObject(value) ? value : {};
}

export function normalizeVlessRealityGrpcConfig(
  config: JsonObject,
  templateType: XrayTemplateType
) {
  const stream = objectValue(config.streamSettings);
  if (
    formString(config.protocol).trim().toLowerCase() !== "vless" ||
    normalizeTransportNetwork(stream.network) !== "grpc" ||
    formString(stream.security).trim().toLowerCase() !== "reality"
  ) {
    return config;
  }

  const settings = objectValue(config.settings);
  if (templateType === "outbound") {
    return {
      ...config,
      settings: { ...settings, flow: "none" },
    };
  }
  if (templateType !== "inbound") return config;

  const normalizedSettings = omitKeys(settings, ["flow"]);
  for (const field of ["clients", "users"] as const) {
    const users = settings[field];
    if (!Array.isArray(users)) continue;
    normalizedSettings[field] = users.map((user) =>
      isJsonObject(user) ? { ...user, flow: "none" } : user
    );
  }

  return {
    ...config,
    settings: normalizedSettings,
  };
}

function firstArrayValue(...values: unknown[]) {
  const arrays = values.filter(Array.isArray);
  return arrays.find((value) => value.length > 0) || arrays[0] || [];
}

function formString(value: unknown) {
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return "";
}

function formBool(value: unknown) {
  return value === true;
}

function formNumber(value: unknown, fallback?: number) {
  const number = numberToUndefined(value);
  return number ?? fallback;
}

export function safeJsonParse<T>(value: string, fallback: T): T {
  if (!value.trim()) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

export function formatJson(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function hasAdvancedJson(value?: unknown) {
  const text = formString(value);
  if (!text.trim()) return false;
  const parsed = safeJsonParse(text, undefined);
  if (parsed === undefined || parsed === null) return false;
  if (
    typeof parsed === "object" &&
    !Array.isArray(parsed) &&
    Object.keys(parsed).length === 0
  ) {
    return false;
  }
  return true;
}

export function linesToArray(value?: string) {
  return (value || "")
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function arrayToLines(value?: unknown) {
  return Array.isArray(value) ? value.join("\n") : "";
}

export function splitCsv(value?: string) {
  return (value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function joinCsv(value?: unknown) {
  if (typeof value === "string") return value;
  return Array.isArray(value) ? value.join(", ") : "";
}

function emptyToUndefined(value: unknown) {
  return value === "" || value === null ? undefined : value;
}

function numberToUndefined(value: unknown) {
  if (value === "" || value === null || value === undefined) return;
  const number = Number(value);
  return Number.isNaN(number) ? undefined : number;
}

function mergeSettings(generated: JsonObject, extraJson?: unknown): JsonObject {
  const extra = safeJsonParse<JsonObject>(formString(extraJson), {});
  return compactObject({ ...generated, ...extra });
}

function omitKeys(value: JsonObject, keys: string[]) {
  const result = { ...value };
  for (const key of keys) {
    delete result[key];
  }
  return result;
}

function nonEmptyObject(value: JsonObject) {
  return Object.keys(value).length ? value : undefined;
}

function streamSettingsExtra(stream: JsonObject) {
  const extra = omitKeys(stream, [
    "network",
    "security",
    "rawSettings",
    "kcpSettings",
    "hysteriaSettings",
    "sockopt",
    "finalmask",
    "wsSettings",
    "grpcSettings",
    "xhttpSettings",
    "httpupgradeSettings",
    "tlsSettings",
    "realitySettings",
  ]);

  const ws = objectValue(stream.wsSettings);
  const wsHeaders = omitKeys(objectValue(ws.headers), ["Host", "host"]);
  const wsExtra = omitKeys(ws, ["path", "host", "headers"]);
  if (Object.keys(wsHeaders).length) wsExtra.headers = wsHeaders;

  const grpcExtra = omitKeys(objectValue(stream.grpcSettings), [
    "serviceName",
    "authority",
    "multiMode",
    "user_agent",
    "idle_timeout",
    "health_check_timeout",
    "permit_without_stream",
    "initial_windows_size",
  ]);
  const xhttpExtra = omitKeys(objectValue(stream.xhttpSettings), [
    "path",
    "host",
    "mode",
    "extra",
  ]);
  const httpUpgradeExtra = omitKeys(objectValue(stream.httpupgradeSettings), [
    "path",
    "host",
  ]);
  const tlsExtra = omitKeys(objectValue(stream.tlsSettings), [
    "certificates",
    "serverName",
    "allowInsecure",
    "fingerprint",
    "pinnedPeerCertSha256",
    "verifyPeerCertByName",
  ]);
  const realityExtra = omitKeys(objectValue(stream.realitySettings), [
    "show",
    "target",
    "dest",
    "xver",
    "serverNames",
    "privateKey",
    "minClientVer",
    "maxClientVer",
    "maxTimeDiff",
    "shortIds",
    "serverName",
    "fingerprint",
    "password",
    "publicKey",
    "shortId",
    "spiderX",
    "mldsa65Seed",
    "mldsa65Verify",
  ]);

  return compactObject({
    ...extra,
    wsSettings: nonEmptyObject(wsExtra),
    grpcSettings: nonEmptyObject(grpcExtra),
    xhttpSettings: nonEmptyObject(xhttpExtra),
    httpupgradeSettings: nonEmptyObject(httpUpgradeExtra),
    tlsSettings: nonEmptyObject(tlsExtra),
    realitySettings: nonEmptyObject(realityExtra),
  });
}

function hasConfigValue(value: unknown) {
  return value !== undefined && value !== null && value !== "";
}

function copyConfigVariable(target: JsonObject, key: string, value: unknown) {
  if (hasConfigValue(value)) {
    target[key] = value;
  }
}

function firstClientField(settings: JsonObject, key: string) {
  const firstClient = Array.isArray(settings.clients)
    ? settings.clients[0]
    : null;
  return objectValue(firstClient)[key];
}

function extractSettingsVariables(
  target: JsonObject,
  settings: JsonObject,
  protocol: string,
  type?: XrayTemplateType
) {
  const normalizedProtocol =
    type === "inbound" ? normalizeInboundProtocol(protocol) : protocol;
  if (type === "outbound") {
    copyConfigVariable(target, "out_address", settings.address);
    copyConfigVariable(target, "out_port", settings.port);
    copyConfigVariable(target, "out_id", settings.id);
    copyConfigVariable(target, "out_encryption", settings.encryption);
    copyConfigVariable(target, "out_security", settings.security);
    copyConfigVariable(target, "out_flow", settings.flow);
    copyConfigVariable(target, "out_password", settings.password);
    copyConfigVariable(target, "out_user", settings.user);
    copyConfigVariable(target, "out_pass", settings.pass);
    if (protocol === "shadowsocks") {
      copyConfigVariable(target, "ss_method", settings.method);
      copyConfigVariable(target, "ss_password", settings.password);
    }
  } else {
    if (normalizedProtocol === "tunnel") {
      copyConfigVariable(
        target,
        "dokodemo_address",
        settings.rewriteAddress ?? settings.address
      );
      copyConfigVariable(
        target,
        "dokodemo_port",
        settings.rewritePort ?? settings.port
      );
    } else {
      copyConfigVariable(target, "address", settings.address);
    }
    if (normalizedProtocol === "shadowsocks") {
      copyConfigVariable(target, "ss_method", settings.method);
      copyConfigVariable(target, "ss_password", settings.password);
    } else {
      copyConfigVariable(target, "method", settings.method);
      copyConfigVariable(target, "password", settings.password);
    }
    copyConfigVariable(target, "id", settings.id);
    copyConfigVariable(target, "flow", settings.flow);
    if (normalizedProtocol === "vless") {
      copyConfigVariable(target, "flow", firstClientField(settings, "flow"));
    }
  }
  copyConfigVariable(target, "hysteria_version", settings.version);
}

function extractStreamVariables(
  target: JsonObject,
  stream: JsonObject,
  type?: XrayTemplateType
) {
  const network = normalizeTransportNetwork(stream.network);
  copyConfigVariable(target, "transport", network);
  copyConfigVariable(target, "security", stream.security);

  const tls = objectValue(stream.tlsSettings);
  copyConfigVariable(target, "sni", tls.serverName);
  copyConfigVariable(target, "server_name", tls.serverName);
  copyConfigVariable(target, "fingerprint", tls.fingerprint);
  copyConfigVariable(target, "allow_insecure", tls.allowInsecure);
  copyConfigVariable(
    target,
    "pinned_peer_cert_sha256",
    tls.pinnedPeerCertSha256
  );
  copyConfigVariable(
    target,
    "verify_peer_cert_by_name",
    tls.verifyPeerCertByName
  );

  const reality = objectValue(stream.realitySettings);
  if (type === "inbound") {
    copyConfigVariable(target, "reality_target", reality.target);
    copyConfigVariable(target, "reality_private_key", reality.privateKey);
    copyConfigVariable(target, "reality_server_names", reality.serverNames);
    copyConfigVariable(target, "reality_short_ids", reality.shortIds);
    copyConfigVariable(target, "reality_mldsa65_seed", reality.mldsa65Seed);
  }
  if (type === "outbound") {
    copyConfigVariable(target, "reality_public_key", reality.password);
    copyConfigVariable(target, "reality_short_id", reality.shortId);
    copyConfigVariable(target, "reality_mldsa65_verify", reality.mldsa65Verify);
    copyConfigVariable(target, "sni", reality.serverName);
  }

  const ws = objectValue(stream.wsSettings);
  const grpc = objectValue(stream.grpcSettings);
  const xhttp = objectValue(stream.xhttpSettings);
  const hysteria = objectValue(stream.hysteriaSettings);
  const httpupgrade = objectValue(stream.httpupgradeSettings);
  if (network === "websocket") {
    copyConfigVariable(target, "path", ws.path);
    copyConfigVariable(target, "host", ws.host ?? objectValue(ws.headers).Host);
  }
  if (network === "grpc") {
    copyConfigVariable(target, "service_name", grpc.serviceName);
    copyConfigVariable(target, "grpc_service_name", grpc.serviceName);
    copyConfigVariable(target, "grpc_authority", grpc.authority);
    if (type === "outbound") {
      copyConfigVariable(target, "grpc_user_agent", grpc.user_agent);
      copyConfigVariable(target, "grpc_idle_timeout", grpc.idle_timeout);
      copyConfigVariable(
        target,
        "grpc_health_check_timeout",
        grpc.health_check_timeout
      );
      copyConfigVariable(
        target,
        "grpc_permit_without_stream",
        grpc.permit_without_stream
      );
      copyConfigVariable(
        target,
        "grpc_initial_windows_size",
        grpc.initial_windows_size
      );
    }
  }
  if (network === "xhttp") {
    copyConfigVariable(target, "host", xhttp.host);
    copyConfigVariable(target, "path", xhttp.path);
    copyConfigVariable(target, "xhttp_mode", xhttp.mode);
    copyConfigVariable(target, "xhttp_extra", xhttp.extra);
  }
  if (network === "hysteria") {
    copyConfigVariable(target, "hysteria_version", hysteria.version);
    copyConfigVariable(target, "udp_idle_timeout", hysteria.udpIdleTimeout);
    copyConfigVariable(target, "hysteria_masquerade", hysteria.masquerade);
  }
  if (network === "httpupgrade") {
    copyConfigVariable(target, "host", httpupgrade.host);
    copyConfigVariable(target, "path", httpupgrade.path);
  }
}

export function extractTemplateConfigVariables(template: {
  type?: XrayTemplateType;
  config?: JsonObject;
}) {
  const type = template.type;
  const config = objectValue(template.config);
  const variables: JsonObject = {};
  copyConfigVariable(variables, "tag", config.tag);
  copyConfigVariable(variables, "listen", config.listen);
  copyConfigVariable(variables, "port", config.port);
  copyConfigVariable(variables, "protocol", config.protocol);
  copyConfigVariable(variables, "sendThrough", config.sendThrough);

  if (type === "dns") {
    copyConfigVariable(variables, "dns_servers", config.servers);
    copyConfigVariable(variables, "hosts", config.hosts);
    copyConfigVariable(variables, "query_strategy", config.queryStrategy);
    copyConfigVariable(variables, "disable_cache", config.disableCache);
    copyConfigVariable(variables, "serve_stale", config.serveStale);
    copyConfigVariable(variables, "serve_expired_ttl", config.serveExpiredTTL);
    copyConfigVariable(variables, "disable_fallback", config.disableFallback);
    copyConfigVariable(
      variables,
      "disable_fallback_if_match",
      config.disableFallbackIfMatch
    );
    copyConfigVariable(
      variables,
      "enable_parallel_query",
      config.enableParallelQuery
    );
    copyConfigVariable(variables, "use_system_hosts", config.useSystemHosts);
    copyConfigVariable(variables, "client_ip", config.clientIp);
    return variables;
  }

  if (type === "routing") {
    copyConfigVariable(
      variables,
      "routing_domain_strategy",
      config.domainStrategy
    );
    copyConfigVariable(variables, "routing_rules", config.rules);
    copyConfigVariable(variables, "routing_balancers", config.balancers);
    return variables;
  }

  if (type === "geodata") {
    copyConfigVariable(variables, "geodata_cron", config.cron);
    copyConfigVariable(variables, "geodata_outbound", config.outbound);
    copyConfigVariable(variables, "geodata_assets", config.assets);
    return variables;
  }

  if (isJsonObject(config.settings)) {
    extractSettingsVariables(
      variables,
      config.settings,
      typeof config.protocol === "string" ? config.protocol : "",
      type
    );
  }
  if (isJsonObject(config.streamSettings)) {
    extractStreamVariables(variables, config.streamSettings, type);
  }
  return variables;
}

function settingsExtra(
  type: XrayTemplateType,
  protocol: string,
  settings: JsonObject
) {
  const normalizedProtocol =
    type === "inbound" ? normalizeInboundProtocol(protocol) : protocol;
  if (type === "inbound") {
    if (normalizedProtocol === "vless") {
      return omitKeys(settings, [
        "clients",
        "users",
        "decryption",
        "fallbacks",
      ]);
    }
    if (normalizedProtocol === "trojan") {
      return omitKeys(settings, ["clients", "users", "fallbacks"]);
    }
    if (normalizedProtocol === "vmess") {
      return omitKeys(settings, ["clients", "users", "default"]);
    }
    if (normalizedProtocol === "shadowsocks") {
      return omitKeys(settings, [
        "network",
        "method",
        "password",
        "level",
        "email",
        "clients",
        "users",
      ]);
    }
    if (normalizedProtocol === "socks") {
      return omitKeys(settings, [
        "auth",
        "accounts",
        "users",
        "udp",
        "ip",
        "userLevel",
      ]);
    }
    if (normalizedProtocol === "http") {
      return omitKeys(settings, [
        "accounts",
        "users",
        "allowTransparent",
        "userLevel",
      ]);
    }
    if (normalizedProtocol === "tunnel") {
      return omitKeys(settings, [
        "address",
        "port",
        "portMap",
        "network",
        "followRedirect",
        "userLevel",
      ]);
    }
    if (normalizedProtocol === "wireguard") {
      return omitKeys(settings, ["secretKey", "peers", "mtu"]);
    }
    if (normalizedProtocol === "hysteria") {
      return omitKeys(settings, ["version", "clients", "users"]);
    }
    if (normalizedProtocol === "tun") {
      return omitKeys(settings, ["name", "MTU", "userLevel", "UserLevel"]);
    }
  }

  if (type === "outbound") {
    if (protocol === "freedom") {
      return omitKeys(settings, [
        "domainStrategy",
        "redirect",
        "userLevel",
        "fragment",
        "noises",
        "proxyProtocol",
      ]);
    }
    if (protocol === "blackhole") {
      return omitKeys(settings, ["response"]);
    }
    if (protocol === "vless") {
      return omitKeys(settings, [
        "address",
        "port",
        "id",
        "encryption",
        "flow",
        "level",
      ]);
    }
    if (protocol === "vmess") {
      return omitKeys(settings, [
        "address",
        "port",
        "id",
        "security",
        "level",
        "experiments",
      ]);
    }
    if (protocol === "trojan") {
      return omitKeys(settings, [
        "address",
        "port",
        "password",
        "email",
        "level",
      ]);
    }
    if (protocol === "shadowsocks") {
      return omitKeys(settings, [
        "address",
        "port",
        "method",
        "password",
        "level",
        "email",
        "uot",
        "UoTVersion",
      ]);
    }
    if (protocol === "socks") {
      return omitKeys(settings, [
        "address",
        "port",
        "user",
        "pass",
        "level",
        "email",
      ]);
    }
    if (protocol === "http") {
      return omitKeys(settings, [
        "address",
        "port",
        "user",
        "pass",
        "level",
        "email",
        "headers",
      ]);
    }
    if (protocol === "dns") {
      return omitKeys(settings, [
        "network",
        "address",
        "port",
        "userLevel",
        "rules",
      ]);
    }
    if (protocol === "loopback") {
      return omitKeys(settings, ["inboundTag"]);
    }
    if (protocol === "wireguard") {
      return omitKeys(settings, [
        "secretKey",
        "address",
        "peers",
        "noKernelTun",
        "mtu",
        "reserved",
        "workers",
        "domainStrategy",
      ]);
    }
    if (protocol === "hysteria") {
      return omitKeys(settings, ["version", "address", "port"]);
    }
  }

  return settings;
}

export function compactObject<T extends JsonObject>(value: T): T {
  const result: JsonObject = {};
  for (const [key, item] of Object.entries(value)) {
    if (item === undefined || item === null || item === "") continue;
    if (Array.isArray(item) && item.length === 0) continue;
    if (
      typeof item === "object" &&
      !Array.isArray(item) &&
      Object.keys(item).length === 0
    ) {
      continue;
    }
    result[key] = item;
  }
  return result as T;
}

export function buildStreamSettings(values: XrayFormValues) {
  if (!values.network && (!values.security || values.security === "none")) {
    return;
  }

  const network = normalizeTransportNetwork(values.network) || "raw";
  const security =
    network === "hysteria" ? "tls" : formString(values.security) || "none";
  const streamExtra = safeJsonParse<JsonObject>(
    formString(values.stream_extra_json),
    {}
  );
  const rootExtra = omitKeys(streamExtra, [
    "wsSettings",
    "grpcSettings",
    "xhttpSettings",
    "httpupgradeSettings",
    "tlsSettings",
    "realitySettings",
  ]);
  const stream: JsonObject = compactObject({
    ...rootExtra,
    network,
    security: security !== "none" ? security : undefined,
  });

  if (network === "websocket") {
    const wsExtra = objectValue(streamExtra.wsSettings);
    stream.wsSettings = compactObject({
      ...wsExtra,
      path: emptyToUndefined(values.path),
      host: emptyToUndefined(values.host),
    });
  }

  if (network === "grpc") {
    const isOutbound = values.type === "outbound";
    stream.grpcSettings = compactObject({
      ...objectValue(streamExtra.grpcSettings),
      serviceName: emptyToUndefined(values.service_name),
      authority: emptyToUndefined(values.grpc_authority),
      multiMode: isOutbound && values.grpc_multi_mode ? true : undefined,
      user_agent: isOutbound
        ? emptyToUndefined(values.grpc_user_agent)
        : undefined,
      idle_timeout: isOutbound
        ? numberToUndefined(values.grpc_idle_timeout)
        : undefined,
      health_check_timeout: isOutbound
        ? numberToUndefined(values.grpc_health_check_timeout)
        : undefined,
      permit_without_stream:
        isOutbound && values.grpc_permit_without_stream ? true : undefined,
      initial_windows_size: isOutbound
        ? numberToUndefined(values.grpc_initial_windows_size)
        : undefined,
    });
  }

  if (network === "xhttp") {
    stream.xhttpSettings = compactObject({
      ...objectValue(streamExtra.xhttpSettings),
      path: emptyToUndefined(values.path),
      host: emptyToUndefined(values.host),
      mode: emptyToUndefined(values.xhttp_mode),
      extra: safeJsonParse(formString(values.xhttp_extra_json), {}),
    });
  }

  if (network === "httpupgrade") {
    stream.httpupgradeSettings = compactObject({
      ...objectValue(streamExtra.httpupgradeSettings),
      path: emptyToUndefined(values.path),
      host: emptyToUndefined(values.host),
    });
  }

  if (network === "raw") {
    stream.rawSettings = safeJsonParse(
      formString(values.raw_settings_json),
      {}
    );
  }

  if (network === "mkcp") {
    stream.kcpSettings = compactObject({
      ...safeJsonParse(formString(values.kcp_settings_json), {}),
      mtu: numberToUndefined(values.kcp_mtu),
      tti: numberToUndefined(values.kcp_tti),
      uplinkCapacity: numberToUndefined(values.kcp_uplink_capacity),
      downlinkCapacity: numberToUndefined(values.kcp_downlink_capacity),
      congestion: values.kcp_congestion || undefined,
      header: values.kcp_header_type
        ? { type: values.kcp_header_type }
        : undefined,
    });
  }

  if (network === "hysteria") {
    const masquerade = safeJsonParse(
      formString(values.hysteria_masquerade_json),
      {}
    );
    stream.hysteriaSettings = compactObject({
      ...safeJsonParse(formString(values.hysteria_settings_json), {}),
      version: numberToUndefined(values.hysteria_version) || 2,
      auth: emptyToUndefined(values.hysteria_auth),
      udpIdleTimeout: numberToUndefined(values.udp_idle_timeout),
      masquerade: Object.keys(masquerade).length ? masquerade : undefined,
    });
  }

  if (security === "tls") {
    stream.tlsSettings = compactObject({
      ...objectValue(streamExtra.tlsSettings),
      ...(values.tls_certificates_json !== undefined
        ? {
            certificates: safeJsonParse(
              formString(values.tls_certificates_json),
              []
            ),
          }
        : {}),
      serverName: emptyToUndefined(values.sni),
      allowInsecure: values.allow_insecure || undefined,
      pinnedPeerCertSha256: emptyToUndefined(values.pinned_peer_cert_sha256),
      verifyPeerCertByName: emptyToUndefined(values.verify_peer_cert_by_name),
      fingerprint: emptyToUndefined(values.fingerprint),
    });
  }

  if (security === "reality") {
    const isOutbound = values.type === "outbound";
    stream.realitySettings = isOutbound
      ? compactObject({
          ...objectValue(streamExtra.realitySettings),
          serverName: emptyToUndefined(values.sni),
          fingerprint: emptyToUndefined(values.fingerprint),
          password: emptyToUndefined(values.reality_public_key),
          shortId: emptyToUndefined(values.reality_short_id),
          spiderX: emptyToUndefined(values.reality_spider_x),
          mldsa65Verify: emptyToUndefined(values.reality_mldsa65_verify),
        })
      : compactObject({
          ...objectValue(streamExtra.realitySettings),
          show: values.reality_show || undefined,
          target: emptyToUndefined(values.reality_target),
          xver: numberToUndefined(values.reality_xver),
          serverNames: splitCsv(formString(values.reality_server_names)),
          privateKey: emptyToUndefined(values.reality_private_key),
          shortIds: splitCsv(formString(values.reality_short_ids)),
          mldsa65Seed: emptyToUndefined(values.reality_mldsa65_seed),
          minClientVer: emptyToUndefined(values.reality_min_client_ver),
          maxClientVer: emptyToUndefined(values.reality_max_client_ver),
          maxTimeDiff: numberToUndefined(values.reality_max_time_diff),
        });
  }

  const sockopt = safeJsonParse(formString(values.sockopt_json), {});
  if (Object.keys(sockopt).length) {
    stream.sockopt = sockopt;
  }
  const finalmask = safeJsonParse(formString(values.finalmask_json), {});
  if (Object.keys(finalmask).length) {
    stream.finalmask = finalmask;
  }

  return stream;
}

function buildInboundProtocolSettings(values: XrayFormValues) {
  const protocol = normalizeInboundProtocol(values.protocol);
  if (protocol === "vless") {
    return mergeSettings(
      {
        clients: safeJsonParse(formString(values.vless_clients_json), []),
        decryption: values.vless_decryption || "none",
        fallbacks: safeJsonParse(formString(values.fallbacks_json), []),
      },
      values.settings_json
    );
  }
  if (protocol === "vmess") {
    return mergeSettings(
      {
        clients: safeJsonParse(formString(values.vmess_clients_json), []),
        default: safeJsonParse(formString(values.vmess_default_json), {}),
      },
      values.settings_json
    );
  }
  if (protocol === "trojan") {
    return mergeSettings(
      {
        clients: safeJsonParse(formString(values.trojan_clients_json), []),
        fallbacks: safeJsonParse(formString(values.fallbacks_json), []),
      },
      values.settings_json
    );
  }
  if (protocol === "shadowsocks") {
    return mergeSettings(
      {
        network: emptyToUndefined(values.ss_network),
        method: emptyToUndefined(values.ss_method),
        password: emptyToUndefined(values.ss_password),
        level: numberToUndefined(values.user_level),
        email: emptyToUndefined(values.email),
        clients: safeJsonParse(formString(values.ss_clients_json), []),
      },
      values.settings_json
    );
  }
  if (protocol === "socks") {
    return mergeSettings(
      {
        auth: values.socks_auth || "noauth",
        accounts: safeJsonParse(formString(values.accounts_json), []),
        udp: values.socks_udp || undefined,
        ip: emptyToUndefined(values.socks_ip),
        userLevel: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  if (protocol === "http") {
    return mergeSettings(
      {
        accounts: safeJsonParse(formString(values.accounts_json), []),
        allowTransparent: values.allow_transparent || undefined,
        userLevel: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  if (protocol === "tunnel") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.dokodemo_address),
        port: numberToUndefined(values.dokodemo_port),
        portMap: safeJsonParse(formString(values.dokodemo_port_map_json), {}),
        network: emptyToUndefined(values.dokodemo_network),
        followRedirect: values.follow_redirect || undefined,
        userLevel: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  if (protocol === "wireguard") {
    return mergeSettings(
      {
        secretKey: emptyToUndefined(values.wg_secret_key),
        peers: safeJsonParse(formString(values.wg_peers_json), []),
        mtu: numberToUndefined(values.wg_mtu),
      },
      values.settings_json
    );
  }
  if (protocol === "hysteria") {
    return mergeSettings(
      {
        version: numberToUndefined(values.hysteria_version),
        users: safeJsonParse(formString(values.hysteria_users_json), []),
      },
      values.settings_json
    );
  }
  if (protocol === "tun") {
    return mergeSettings(
      {
        name: emptyToUndefined(values.tun_name),
        MTU: numberToUndefined(values.tun_mtu),
        userLevel: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  return safeJsonParse(formString(values.settings_json), {});
}

function buildOutboundProtocolSettings(values: XrayFormValues) {
  const protocol = formString(values.protocol);
  if (protocol === "freedom") {
    return mergeSettings(
      {
        domainStrategy: emptyToUndefined(values.domain_strategy),
        redirect: emptyToUndefined(values.redirect),
        userLevel: numberToUndefined(values.user_level),
        fragment: safeJsonParse(formString(values.fragment_json), {}),
        noises: safeJsonParse(formString(values.noises_json), []),
        proxyProtocol: numberToUndefined(values.proxy_protocol),
      },
      values.settings_json
    );
  }
  if (protocol === "blackhole") {
    return mergeSettings(
      {
        response: { type: values.response_type || "none" },
      },
      values.settings_json
    );
  }
  if (protocol === "vless") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
        id: emptyToUndefined(values.out_id),
        encryption: values.out_encryption || "none",
        flow: emptyToUndefined(values.out_flow),
        level: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  if (protocol === "vmess") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
        id: emptyToUndefined(values.out_id),
        security: values.out_security || "auto",
        level: numberToUndefined(values.user_level),
        experiments: emptyToUndefined(values.out_experiments),
      },
      values.settings_json
    );
  }
  if (protocol === "trojan") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
        password: emptyToUndefined(values.out_password),
        email: emptyToUndefined(values.email),
        level: numberToUndefined(values.user_level),
      },
      values.settings_json
    );
  }
  if (protocol === "shadowsocks") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
        method: emptyToUndefined(values.ss_method),
        password: emptyToUndefined(values.ss_password),
        level: numberToUndefined(values.user_level),
        email: emptyToUndefined(values.email),
        uot: values.ss_uot || undefined,
        UoTVersion: numberToUndefined(values.ss_uot_version),
      },
      values.settings_json
    );
  }
  if (protocol === "socks" || protocol === "http") {
    return mergeSettings(
      {
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
        user: emptyToUndefined(values.out_user),
        pass: emptyToUndefined(values.out_pass),
        level: numberToUndefined(values.user_level),
        email: emptyToUndefined(values.email),
        headers:
          protocol === "http"
            ? safeJsonParse(formString(values.headers_json), {})
            : undefined,
      },
      values.settings_json
    );
  }
  if (protocol === "dns") {
    return mergeSettings(
      {
        network: emptyToUndefined(values.dns_out_network),
        address: emptyToUndefined(values.dns_out_address),
        port: numberToUndefined(values.dns_out_port),
        userLevel: numberToUndefined(values.user_level),
        rules: safeJsonParse(formString(values.dns_out_rules_json), []),
      },
      values.settings_json
    );
  }
  if (protocol === "loopback") {
    return mergeSettings(
      {
        inboundTag: emptyToUndefined(values.loopback_inbound_tag),
      },
      values.settings_json
    );
  }
  if (protocol === "wireguard") {
    return mergeSettings(
      {
        secretKey: emptyToUndefined(values.wg_secret_key),
        address: linesToArray(formString(values.wg_address)),
        peers: safeJsonParse(formString(values.wg_peers_json), []),
        noKernelTun: values.wg_no_kernel_tun ?? false,
        mtu: numberToUndefined(values.wg_mtu),
        reserved: safeJsonParse(formString(values.wg_reserved_json), []),
        workers: numberToUndefined(values.wg_workers),
        domainStrategy: emptyToUndefined(values.wg_domain_strategy),
      },
      values.settings_json
    );
  }
  if (protocol === "hysteria") {
    return mergeSettings(
      {
        version: numberToUndefined(values.hysteria_version),
        address: emptyToUndefined(values.out_address),
        port: numberToUndefined(values.out_port),
      },
      values.settings_json
    );
  }
  return safeJsonParse(formString(values.settings_json), {});
}

export function buildInboundConfig(values: XrayFormValues) {
  const protocol = normalizeInboundProtocol(values.protocol);
  const settings = buildInboundProtocolSettings(values);
  const streamUnsupported = ["tun", "wireguard"].includes(protocol);
  const config = compactObject({
    tag: values.tag,
    listen: values.listen,
    port: values.port ? Number(values.port) : undefined,
    protocol,
    settings,
    streamSettings: streamUnsupported ? undefined : buildStreamSettings(values),
    sniffing: values.sniffing
      ? compactObject({
          enabled: true,
          destOverride: splitCsv(formString(values.dest_override)),
          metadataOnly: values.sniffing_metadata_only || undefined,
          routeOnly: values.sniffing_route_only || undefined,
          domainsExcluded: splitCsv(
            formString(values.sniffing_domains_excluded)
          ),
        })
      : undefined,
  });

  const result = hasAdvancedJson(values.advanced_json)
    ? safeJsonParse(formString(values.advanced_json), config)
    : config;
  return normalizeVlessRealityGrpcConfig(result, "inbound");
}

export function buildOutboundConfig(values: XrayFormValues) {
  const protocol = formString(values.protocol);
  const settings = buildOutboundProtocolSettings(values);
  const streamUnsupported = ["dns", "loopback", "wireguard"].includes(protocol);

  const config = compactObject({
    tag: values.tag,
    protocol,
    settings,
    streamSettings: streamUnsupported ? undefined : buildStreamSettings(values),
  });

  const result = hasAdvancedJson(values.advanced_json)
    ? safeJsonParse(formString(values.advanced_json), config)
    : config;
  return normalizeVlessRealityGrpcConfig(result, "outbound");
}

export function buildDnsConfig(values: XrayFormValues) {
  const objectServers = safeJsonParse<unknown[]>(
    formString(values.dns_servers_json),
    []
  );
  const config = compactObject({
    hosts: safeJsonParse(formString(values.hosts_json), {}),
    servers: [...linesToArray(formString(values.servers)), ...objectServers],
    tag: emptyToUndefined(values.tag),
    clientIp: emptyToUndefined(values.client_ip),
    queryStrategy: values.query_strategy,
    disableCache: values.disable_cache || undefined,
    serveStale: values.serve_stale || undefined,
    serveExpiredTTL: values.serve_expired_ttl
      ? Number(values.serve_expired_ttl)
      : undefined,
    disableFallback: values.disable_fallback || undefined,
    disableFallbackIfMatch: values.disable_fallback_if_match || undefined,
    enableParallelQuery: values.enable_parallel_query || undefined,
    useSystemHosts: values.use_system_hosts || undefined,
  });

  return hasAdvancedJson(values.advanced_json)
    ? safeJsonParse(formString(values.advanced_json), config)
    : config;
}

export function buildRoutingConfig(values: XrayFormValues) {
  const config = compactObject({
    domainStrategy: values.routing_domain_strategy || "AsIs",
    rules: safeJsonParse(formString(values.routing_rules_json), []),
    balancers: safeJsonParse(formString(values.routing_balancers_json), []),
  });

  return hasAdvancedJson(values.advanced_json)
    ? safeJsonParse(formString(values.advanced_json), config)
    : config;
}

export function buildGeodataConfig(values: XrayFormValues) {
  const config = compactObject({
    cron: values.geodata_cron || "0 4 * * *",
    outbound: emptyToUndefined(values.geodata_outbound),
    assets: safeJsonParse(formString(values.geodata_assets_json), []),
  });

  return hasAdvancedJson(values.advanced_json)
    ? safeJsonParse(formString(values.advanced_json), config)
    : config;
}

export function configToFormValues(
  type: XrayTemplateType,
  config?: JsonObject
) {
  const value = objectValue(config);
  const stream = objectValue(value.streamSettings);
  const ws = objectValue(stream.wsSettings);
  const grpc = objectValue(stream.grpcSettings);
  const xhttp = objectValue(stream.xhttpSettings);
  const httpupgrade = objectValue(stream.httpupgradeSettings);
  const tls = objectValue(stream.tlsSettings);
  const reality = objectValue(stream.realitySettings);
  const settings = objectValue(value.settings);
  const kcp = objectValue(stream.kcpSettings);
  const hysteria = objectValue(stream.hysteriaSettings);
  const sniffing = objectValue(value.sniffing);
  const response = objectValue(settings.response);
  const rawProtocol =
    typeof value.protocol === "string"
      ? value.protocol
      : type === "inbound"
        ? "vless"
        : "freedom";
  const protocol =
    type === "inbound" ? normalizeInboundProtocol(rawProtocol) : rawProtocol;
  const network = normalizeTransportNetwork(stream.network) || "raw";

  if (type === "dns") {
    const servers = Array.isArray(value.servers) ? value.servers : [];
    const stringServers = servers.filter((item) => typeof item === "string");
    const objectServers = servers.filter(isJsonObject);
    return {
      tag: formString(value.tag),
      servers: arrayToLines(stringServers),
      dns_servers_json: formatJson(objectServers),
      hosts_json: formatJson(value.hosts || {}),
      query_strategy: formString(value.queryStrategy) || "UseIP",
      disable_cache: formBool(value.disableCache),
      serve_stale: formBool(value.serveStale),
      serve_expired_ttl: numberToUndefined(value.serveExpiredTTL),
      disable_fallback: formBool(value.disableFallback),
      disable_fallback_if_match: formBool(value.disableFallbackIfMatch),
      enable_parallel_query: formBool(value.enableParallelQuery),
      use_system_hosts: formBool(value.useSystemHosts),
      client_ip: formString(value.clientIp),
      advanced_json: formatJson(value),
    };
  }

  if (type === "routing") {
    return {
      routing_domain_strategy: formString(value.domainStrategy) || "AsIs",
      routing_rules_json: formatJson(value.rules || []),
      routing_balancers_json: formatJson(value.balancers || []),
      advanced_json: formatJson(value),
    };
  }

  if (type === "geodata") {
    return {
      geodata_cron: formString(value.cron) || "0 4 * * *",
      geodata_outbound: formString(value.outbound),
      geodata_assets_json: formatJson(value.assets || []),
      advanced_json: formatJson(value),
    };
  }

  return {
    tag: formString(value.tag),
    listen: formString(value.listen),
    port: numberToUndefined(value.port),
    protocol,
    settings_json: formatJson(settingsExtra(type, protocol, settings)),
    network,
    security:
      network === "hysteria" ? "tls" : formString(stream.security) || "none",
    host:
      formString(ws.host) ||
      formString(objectValue(ws.headers).Host) ||
      formString(xhttp.host) ||
      formString(httpupgrade.host),
    path:
      formString(ws.path) ||
      formString(xhttp.path) ||
      formString(httpupgrade.path),
    service_name: formString(grpc.serviceName),
    grpc_authority: formString(grpc.authority),
    grpc_multi_mode: formBool(grpc.multiMode),
    grpc_user_agent: formString(grpc.user_agent),
    grpc_idle_timeout: numberToUndefined(grpc.idle_timeout),
    grpc_health_check_timeout: numberToUndefined(grpc.health_check_timeout),
    grpc_permit_without_stream: formBool(grpc.permit_without_stream),
    grpc_initial_windows_size: numberToUndefined(grpc.initial_windows_size),
    xhttp_mode: formString(xhttp.mode) || "auto",
    xhttp_extra_json: formatJson(xhttp.extra || {}),
    kcp_mtu: numberToUndefined(kcp.mtu),
    kcp_tti: numberToUndefined(kcp.tti),
    kcp_uplink_capacity: numberToUndefined(kcp.uplinkCapacity),
    kcp_downlink_capacity: numberToUndefined(kcp.downlinkCapacity),
    kcp_congestion: formBool(kcp.congestion),
    kcp_header_type: formString(objectValue(kcp.header).type),
    raw_settings_json: formatJson(stream.rawSettings || {}),
    kcp_settings_json: formatJson(
      omitKeys(kcp, [
        "mtu",
        "tti",
        "uplinkCapacity",
        "downlinkCapacity",
        "congestion",
        "header",
      ])
    ),
    hysteria_auth: formString(hysteria.auth),
    udp_idle_timeout: numberToUndefined(hysteria.udpIdleTimeout),
    hysteria_masquerade_json: formatJson(hysteria.masquerade || {}),
    hysteria_settings_json: formatJson(
      omitKeys(hysteria, ["version", "auth", "udpIdleTimeout", "masquerade"])
    ),
    sockopt_json: formatJson(stream.sockopt || {}),
    finalmask_json: formatJson(stream.finalmask || {}),
    stream_extra_json: formatJson(streamSettingsExtra(stream)),
    sni: formString(tls.serverName) || formString(reality.serverName),
    allow_insecure: formBool(tls.allowInsecure),
    tls_certificates_json: formatJson(tls.certificates || []),
    pinned_peer_cert_sha256: formString(tls.pinnedPeerCertSha256),
    verify_peer_cert_by_name: formString(tls.verifyPeerCertByName),
    fingerprint:
      formString(tls.fingerprint) ||
      formString(reality.fingerprint) ||
      "chrome",
    reality_show: formBool(reality.show),
    reality_target: formString(reality.target) || formString(reality.dest),
    reality_xver: numberToUndefined(reality.xver),
    reality_server_names: joinCsv(reality.serverNames || reality.serverName),
    reality_private_key: formString(reality.privateKey),
    reality_public_key:
      formString(reality.password) || formString(reality.publicKey),
    reality_short_id: formString(reality.shortId),
    reality_short_ids: joinCsv(reality.shortIds),
    reality_spider_x: formString(reality.spiderX),
    reality_mldsa65_seed: formString(reality.mldsa65Seed),
    reality_mldsa65_verify: formString(reality.mldsa65Verify),
    reality_min_client_ver: formString(reality.minClientVer),
    reality_max_client_ver: formString(reality.maxClientVer),
    reality_max_time_diff: numberToUndefined(reality.maxTimeDiff),
    sniffing: formBool(sniffing.enabled),
    dest_override: joinCsv(sniffing.destOverride || ["http", "tls"]),
    sniffing_metadata_only: formBool(sniffing.metadataOnly),
    sniffing_route_only: formBool(sniffing.routeOnly),
    sniffing_domains_excluded: joinCsv(sniffing.domainsExcluded || []),
    vless_clients_json: formatJson(
      protocol === "vless"
        ? firstArrayValue(settings.clients, settings.users)
        : []
    ),
    vless_decryption: formString(settings.decryption) || "none",
    vmess_clients_json: formatJson(
      protocol === "vmess"
        ? firstArrayValue(settings.clients, settings.users)
        : []
    ),
    vmess_default_json: formatJson(
      protocol === "vmess" ? settings.default || {} : {}
    ),
    trojan_clients_json: formatJson(
      protocol === "trojan"
        ? firstArrayValue(settings.clients, settings.users)
        : []
    ),
    fallbacks_json: formatJson(settings.fallbacks || []),
    ss_network: formString(settings.network) || "tcp",
    ss_method: formString(settings.method),
    ss_password: formString(settings.password),
    ss_clients_json: formatJson(
      protocol === "shadowsocks"
        ? firstArrayValue(settings.clients, settings.users)
        : []
    ),
    socks_auth: formString(settings.auth) || "noauth",
    socks_udp: formBool(settings.udp),
    socks_ip: formString(settings.ip),
    accounts_json: formatJson(
      ["socks", "http"].includes(protocol)
        ? firstArrayValue(settings.accounts, settings.users)
        : []
    ),
    allow_transparent: formBool(settings.allowTransparent),
    dokodemo_address:
      formString(settings.rewriteAddress) || formString(settings.address),
    dokodemo_port:
      numberToUndefined(settings.rewritePort) ??
      numberToUndefined(settings.port),
    dokodemo_network:
      formString(settings.allowedNetwork) ||
      formString(settings.network) ||
      "tcp",
    follow_redirect: formBool(settings.followRedirect),
    dokodemo_port_map_json: formatJson(settings.portMap || {}),
    wg_secret_key: formString(settings.secretKey),
    wg_address: arrayToLines(settings.address || []),
    wg_peers_json: formatJson(settings.peers || []),
    wg_mtu: formNumber(settings.mtu, 1420),
    wg_no_kernel_tun: formBool(settings.noKernelTun),
    wg_reserved_json: formatJson(settings.reserved || []),
    wg_workers: numberToUndefined(settings.workers),
    wg_domain_strategy: formString(settings.domainStrategy) || "ForceIP",
    hysteria_version: formNumber(settings.version, 2),
    hysteria_users_json: formatJson(
      protocol === "hysteria"
        ? firstArrayValue(settings.users, settings.clients)
        : []
    ),
    tun_name: formString(settings.name) || "xray0",
    tun_mtu: formNumber(settings.mtu) ?? formNumber(settings.MTU, 1500),
    out_address: formString(settings.address),
    out_port: numberToUndefined(settings.port),
    out_id: formString(settings.id),
    out_encryption: formString(settings.encryption) || "none",
    out_flow: formString(settings.flow),
    out_security: formString(settings.security) || "auto",
    out_experiments: formString(settings.experiments),
    out_password: formString(settings.password),
    out_user: formString(settings.user),
    out_pass: formString(settings.pass),
    ss_uot: formBool(settings.uot),
    ss_uot_version:
      formNumber(settings.UoTVersion) ?? formNumber(settings.uotVersion, 2),
    headers_json: formatJson(settings.headers || {}),
    email: formString(settings.email),
    user_level:
      formNumber(settings.userLevel) ?? formNumber(settings.level, undefined),
    domain_strategy: formString(settings.domainStrategy),
    redirect: formString(settings.redirect),
    fragment_json: formatJson(settings.fragment || {}),
    noises_json: formatJson(settings.noises || []),
    proxy_protocol: numberToUndefined(settings.proxyProtocol),
    response_type: formString(response.type) || "none",
    dns_out_network: formString(settings.network),
    dns_out_address: formString(settings.address),
    dns_out_port: numberToUndefined(settings.port),
    dns_out_rules_json: formatJson(settings.rules || []),
    loopback_inbound_tag: formString(settings.inboundTag),
    advanced_json: formatJson(value),
  };
}
