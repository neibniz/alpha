export type XrayTemplateType = API.XrayTemplateListItem["type"];
export type XrayTemplateTypeFilter = XrayTemplateType | "all";
export type XrayTemplateStateFilter = "all" | "enabled" | "disabled";
export type XrayTemplateBindingFilter = "all" | "bound" | "unbound";
export type XrayTemplateUseState = "ready" | "unused" | "disabled";

export type XrayTemplateFilters = {
  search: string;
  type: XrayTemplateTypeFilter;
  state: XrayTemplateStateFilter;
  binding: XrayTemplateBindingFilter;
};

const TYPE_VALUES = new Set<XrayTemplateType>([
  "inbound",
  "outbound",
  "dns",
  "routing",
  "geodata",
]);

const CAPABILITY_PATTERNS: [string, RegExp][] = [
  ["VLESS", /\bvless\b/i],
  ["VMess", /\bvmess\b/i],
  ["Trojan", /\btrojan\b/i],
  ["Hysteria2", /\bhysteria\s*2\b|\bhy2\b/i],
  ["Shadowsocks", /\bshadowsocks\b/i],
  ["Reality", /\breality\b/i],
  ["XHTTP", /\bxhttp\b/i],
  ["gRPC", /\bgrpc\b/i],
  ["WebSocket", /\bwebsocket\b|\bws\b/i],
  ["TCP", /\btcp\b/i],
  ["Freedom", /\bfreedom\b|\bdirect\b|直连/i],
  ["Blackhole", /\bblackhole\b|\bblock\b|阻断/i],
  ["DNS", /\bdns\b/i],
  ["Routing", /\brouting\b|路由/i],
  ["GeoData", /\bgeodata\b|\bgeoip\b|\bgeosite\b|地理数据/i],
];

const TYPE_FALLBACKS: Record<XrayTemplateType, string> = {
  inbound: "Inbound",
  outbound: "Outbound",
  dns: "DNS",
  routing: "Routing",
  geodata: "GeoData",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function optionalObject(
  value: unknown,
  field: string,
  index: number
): API.JsonObject | undefined {
  if (value === undefined || value === null) return;
  if (!isRecord(value)) {
    throw new Error(`模板 ${index + 1} 的 ${field} 必须是 JSON 对象`);
  }
  return value as API.JsonObject;
}

function optionalString(value: unknown) {
  return typeof value === "string" ? value : undefined;
}

export function getXrayTemplateCapabilities(
  template: Pick<API.XrayTemplateListItem, "name" | "description" | "type">
) {
  const haystack = `${template.name} ${template.description}`;
  const labels: string[] = [];
  for (const [label, pattern] of CAPABILITY_PATTERNS) {
    if (pattern.test(haystack)) labels.push(label);
    if (labels.length === 3) break;
  }
  return labels.length > 0 ? labels : [TYPE_FALLBACKS[template.type]];
}

export function getXrayTemplateUseState(
  template: Pick<API.XrayTemplateListItem, "enabled" | "binding_count">
): XrayTemplateUseState {
  if (!template.enabled) return "disabled";
  return template.binding_count > 0 ? "ready" : "unused";
}

export function buildXrayTemplateSummary(
  templates: API.XrayTemplateListItem[]
) {
  let enabled = 0;
  let attention = 0;
  let bindings = 0;
  let subscriptions = 0;

  for (const template of templates) {
    if (template.enabled) enabled += 1;
    if (getXrayTemplateUseState(template) === "unused") attention += 1;
    bindings += Math.max(0, template.binding_count || 0);
    subscriptions += Math.max(0, template.subscription_binding_count || 0);
  }

  return {
    total: templates.length,
    enabled,
    attention,
    bindings,
    subscriptions,
  };
}

export function filterXrayTemplates(
  templates: API.XrayTemplateListItem[],
  filters: XrayTemplateFilters
) {
  const search = filters.search.trim().toLocaleLowerCase();
  return templates.filter((template) => {
    const matchesSearch =
      !search ||
      [
        template.name,
        template.description,
        String(template.id),
        ...getXrayTemplateCapabilities(template),
      ].some((value) => value.toLocaleLowerCase().includes(search));
    const matchesType =
      filters.type === "all" || template.type === filters.type;
    const matchesState =
      filters.state === "all" ||
      (filters.state === "enabled" && template.enabled) ||
      (filters.state === "disabled" && !template.enabled);
    const matchesBinding =
      filters.binding === "all" ||
      (filters.binding === "bound" && template.binding_count > 0) ||
      (filters.binding === "unbound" && template.binding_count <= 0);
    return matchesSearch && matchesType && matchesState && matchesBinding;
  });
}

export function xrayTemplateDetailToCreateRequest(
  template: API.XrayTemplate,
  overrides: Partial<
    Pick<API.CreateXrayTemplateRequest, "name" | "enabled">
  > = {}
): API.CreateXrayTemplateRequest {
  return {
    name: overrides.name ?? template.name,
    type: template.type,
    config: template.config || {},
    config_template: template.config_template,
    variables_schema: template.variables_schema,
    default_variables: template.default_variables,
    subscription_meta: template.subscription_meta,
    description: template.description,
    enabled: overrides.enabled ?? template.enabled,
  };
}

export function normalizeImportedXrayTemplates(
  input: unknown
): API.CreateXrayTemplateRequest[] {
  const candidates = Array.isArray(input)
    ? input
    : isRecord(input) && Array.isArray(input.templates)
      ? input.templates
      : [input];

  if (candidates.length === 0) throw new Error("导入文件中没有模板");
  if (candidates.length > 50) throw new Error("单次最多导入 50 个模板");

  return candidates.map((candidate, index) => {
    if (!isRecord(candidate)) {
      throw new Error(`模板 ${index + 1} 必须是 JSON 对象`);
    }
    const name =
      typeof candidate.name === "string" ? candidate.name.trim() : "";
    if (!name) throw new Error(`模板 ${index + 1} 缺少名称`);
    if (!TYPE_VALUES.has(candidate.type as XrayTemplateType)) {
      throw new Error(`模板 ${index + 1} 的类型无效`);
    }
    const config = optionalObject(candidate.config, "config", index) || {};
    return {
      name,
      type: candidate.type as XrayTemplateType,
      config,
      config_template: optionalString(candidate.config_template),
      variables_schema: optionalObject(
        candidate.variables_schema,
        "variables_schema",
        index
      ),
      default_variables: optionalObject(
        candidate.default_variables,
        "default_variables",
        index
      ),
      subscription_meta: optionalObject(
        candidate.subscription_meta,
        "subscription_meta",
        index
      ),
      description: optionalString(candidate.description),
      enabled:
        typeof candidate.enabled === "boolean" ? candidate.enabled : false,
    };
  });
}
