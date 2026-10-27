import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Switch } from "@workspace/ui/components/switch";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Textarea } from "@workspace/ui/components/textarea";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { cn } from "@workspace/ui/lib/utils";
import {
  bindServerXrayTemplates,
  previewServerXrayConfig,
  queryServerXrayTemplateList,
  queryXrayTemplateCatalog,
} from "@workspace/ui/services/admin/server";
import { normalizeListResult } from "@workspace/ui/utils/pagination";
import { Braces, CircleCheck, TriangleAlert } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  extractTemplateConfigVariables,
  formatJson,
  safeJsonParse,
  type XrayTemplateType,
} from "./config";
import { JsonObjectEditor, parseJsonObjectText } from "./json-form-controls";
import {
  buildBindingByTemplateId,
  findBindingIndexByTemplateId,
  groupRenderedTemplates,
  selectEnabledTemplateBindings,
} from "./server-bind-form-data";
import { getXrayFieldHelp, XrayFieldHelp } from "./xray-field-help";

type BindingRow = {
  template_id: number;
  sort: number;
  enabled: boolean;
  alias: string;
  variables_json: string;
  subscription_enabled: boolean;
  subscription_name: string;
  subscription_variables_json: string;
};

type VariableHint = {
  key: string;
  type?: string;
  title?: string;
  description?: string;
  defaultValue?: unknown;
  configValue?: unknown;
  templateDefaultValue?: unknown;
  required?: boolean;
};

type JsonObject = Record<string, unknown>;
type BindXrayTemplate = API.XrayTemplateCatalogItem;

type RenderedTemplate = {
  id: number;
  name: string;
  type: XrayTemplateType;
  alias: string;
  sort: number;
  variables: JsonObject;
  config?: JsonObject;
  raw?: string;
  error?: string;
};

type PreviewIssue = {
  message: string;
};

type BindTab = XrayTemplateType | "preview" | "variables";

const TYPE_LABELS: Record<XrayTemplateType, string> = {
  inbound: "Inbound",
  outbound: "Outbound",
  dns: "DNS",
  routing: "Routing",
  geodata: "Geodata",
};

const TYPE_ORDER: XrayTemplateType[] = [
  "inbound",
  "outbound",
  "dns",
  "routing",
  "geodata",
];

const TYPE_ACCENTS: Record<XrayTemplateType, string> = {
  inbound: "border-blue-200 bg-blue-50 text-blue-700",
  outbound: "border-emerald-200 bg-emerald-50 text-emerald-700",
  dns: "border-cyan-200 bg-cyan-50 text-cyan-700",
  routing: "border-amber-200 bg-amber-50 text-amber-700",
  geodata: "border-violet-200 bg-violet-50 text-violet-700",
};

const VARIABLE_SOURCE_STYLES = {
  override: "border-blue-200 bg-blue-50 text-blue-700",
  config: "border-emerald-200 bg-emerald-50 text-emerald-700",
  default: "border-amber-200 bg-amber-50 text-amber-700",
  empty: "border-muted bg-muted/40 text-muted-foreground",
} as const;

type BindingStatusKey =
  | "effective"
  | "draft"
  | "pending"
  | "applying"
  | "failed"
  | "invalid"
  | "offline"
  | "disabled"
  | "unknown";

type BindingStatusView = {
  key: BindingStatusKey;
  label: string;
  description: string;
};

const BINDING_STATUS_STYLES: Record<
  BindingStatusKey,
  { badge: string; card: string; dot: string }
> = {
  effective: {
    badge: "border-emerald-200 bg-emerald-50 text-emerald-700",
    card: "border-emerald-200 bg-emerald-50/45",
    dot: "bg-emerald-500",
  },
  draft: {
    badge: "border-amber-200 bg-amber-50 text-amber-700",
    card: "border-amber-200 bg-amber-50/40",
    dot: "bg-amber-500",
  },
  pending: {
    badge: "border-blue-200 bg-blue-50 text-blue-700",
    card: "border-blue-200 bg-blue-50/40",
    dot: "bg-blue-500",
  },
  applying: {
    badge: "border-violet-200 bg-violet-50 text-violet-700",
    card: "border-violet-200 bg-violet-50/40",
    dot: "bg-violet-500",
  },
  failed: {
    badge: "border-red-200 bg-red-50 text-red-700",
    card: "border-red-200 bg-red-50/45",
    dot: "bg-red-500",
  },
  invalid: {
    badge: "border-orange-200 bg-orange-50 text-orange-700",
    card: "border-orange-200 bg-orange-50/45",
    dot: "bg-orange-500",
  },
  offline: {
    badge: "border-slate-200 bg-slate-50 text-slate-600",
    card: "border-slate-200 bg-slate-50/60",
    dot: "bg-slate-400",
  },
  disabled: {
    badge: "border-muted bg-muted/60 text-muted-foreground",
    card: "border-muted bg-muted/30",
    dot: "bg-muted-foreground/50",
  },
  unknown: {
    badge: "border-muted bg-background text-muted-foreground",
    card: "border-muted bg-background",
    dot: "bg-muted-foreground/60",
  },
};

function isReferenceSafeAlias(value: string) {
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(value);
}

function referenceExample(type: XrayTemplateType, alias: string) {
  if (!alias) return `{{ .Ref.${type}.main.tag }}`;
  if (isReferenceSafeAlias(alias)) return `{{ .Ref.${type}.${alias}.tag }}`;
  return `{{ (index .Ref.${type} "${alias}").tag }}`;
}

function stringifyHintValue(value: unknown) {
  if (value === undefined || value === null || value === "") return "";
  if (typeof value === "string") return value;
  return JSON.stringify(value);
}

function isJsonObject(value: unknown): value is JsonObject {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function objectValue(value: unknown): JsonObject {
  return isJsonObject(value) ? value : {};
}

function hasOwnValue(source: JsonObject, key: string) {
  return Object.hasOwn(source, key);
}

function inferHintType(value: unknown) {
  if (Array.isArray(value)) return "array";
  if (typeof value === "boolean") return "boolean";
  if (typeof value === "number") return "number";
  if (value && typeof value === "object") return "object";
  return "string";
}

function compactValue(value: unknown) {
  const text = stringifyHintValue(value);
  if (!text) return "未设置";
  return text.length > 96 ? `${text.slice(0, 96)}...` : text;
}

function templateDescription(template: BindXrayTemplate) {
  if (template.description) return template.description;
  const tag = template.config?.tag;
  return typeof tag === "string" && tag ? tag : "—";
}

function inheritedVariables(template: BindXrayTemplate) {
  return {
    ...objectValue(template.default_variables),
    ...extractTemplateConfigVariables({
      type: template.type as XrayTemplateType,
      config: objectValue(template.config),
    }),
  };
}

function collectVariableHints(template: BindXrayTemplate): VariableHint[] {
  const schema = objectValue(template.variables_schema);
  const defaults = objectValue(template.default_variables);
  const configDefaults = extractTemplateConfigVariables({
    type: template.type as XrayTemplateType,
    config: objectValue(template.config),
  });
  const required = Array.isArray(schema.required) ? schema.required : [];
  const properties = isJsonObject(schema.properties)
    ? schema.properties
    : schema;

  const keys = new Set<string>([
    ...Object.keys(configDefaults),
    ...Object.keys(defaults),
    ...Object.keys(properties || {}),
  ]);

  return [...keys].map((key) => {
    const item = isJsonObject(properties?.[key])
      ? objectValue(properties[key])
      : {};
    const configValue = configDefaults[key];
    const templateDefaultValue = defaults[key];
    const inheritedValue = hasOwnValue(configDefaults, key)
      ? configValue
      : hasOwnValue(defaults, key)
        ? templateDefaultValue
        : item.default;
    return {
      key,
      type:
        typeof item.type === "string"
          ? item.type
          : inferHintType(inheritedValue),
      title:
        typeof item.title === "string"
          ? item.title
          : typeof item.label === "string"
            ? item.label
            : undefined,
      description:
        typeof item.description === "string"
          ? item.description
          : typeof item.desc === "string"
            ? item.desc
            : typeof item.help === "string"
              ? item.help
              : undefined,
      defaultValue: inheritedValue,
      configValue,
      templateDefaultValue,
      required: required.includes(key) || item.required === true,
    };
  });
}

function renderSubscriptionHints(template: BindXrayTemplate) {
  const meta = objectValue(template.subscription_meta);
  const keys = Object.keys(meta);
  return (
    <div className="space-y-1 text-muted-foreground text-xs">
      <p>
        订阅变量只用于订阅渲染，可填写每个服务器不同的订阅参数，例如显示名称、
        host、port、sni、transport 或 path。
      </p>
      {keys.length ? (
        <p>
          当前模板订阅元数据字段：{" "}
          <span className="font-mono">{keys.join(", ")}</span>
        </p>
      ) : null}
    </div>
  );
}

function getPathValue(source: JsonObject, path: string[]) {
  return path.reduce<unknown>(
    (current, key) => (isJsonObject(current) ? current[key] : undefined),
    source
  );
}

function templateValueToString(value: unknown) {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

function renderTemplateString(source: string, context: JsonObject): string {
  return source.replace(
    /\{\{\s*(toJson\s+)?\.([A-Za-z]+)((?:\.[A-Za-z0-9_-]+)+)\s*\}\}/g,
    (_match, toJson: string | undefined, root: string, pathText: string) => {
      const rootKey = root.toLowerCase();
      const value = getPathValue(
        objectValue(context[rootKey]),
        pathText.split(".").filter(Boolean)
      );
      return toJson
        ? JSON.stringify(value ?? null)
        : templateValueToString(value);
    }
  );
}

function mergeDnsObjects(items: JsonObject[]) {
  if (!items.length) return;
  if (items.length === 1) return items[0];
  const merged: JsonObject = {};
  for (const item of items) {
    Object.assign(merged, item);
    merged.servers = [...toArray(merged.servers), ...toArray(item.servers)];
    merged.hosts = { ...objectValue(merged.hosts), ...objectValue(item.hosts) };
  }
  return merged;
}

function mergeRoutingObjects(items: JsonObject[]) {
  if (!items.length) return;
  if (items.length === 1) return items[0];
  const merged: JsonObject = {};
  for (const item of items) {
    Object.assign(merged, item);
    merged.rules = appendUniqueObjects(merged.rules, item.rules);
    merged.balancers = appendUniqueObjects(merged.balancers, item.balancers);
  }
  return merged;
}

function appendUniqueObjects(current?: unknown, next?: unknown) {
  const result: unknown[] = [];
  const seen = new Set<string>();
  for (const item of [...toArray(current), ...toArray(next)]) {
    const key = stableJsonKey(item);
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(item);
  }
  return result;
}

function toArray(value?: unknown) {
  if (Array.isArray(value)) return value;
  return value == null ? [] : [value];
}

function stableJsonKey(value: unknown): string {
  return JSON.stringify(sortJsonValue(value));
}

function sortJsonValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortJsonValue);
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, entry]) => [key, sortJsonValue(entry)])
    );
  }
  return value;
}

function templateTag(template: BindXrayTemplate) {
  const config = objectValue(template.config);
  const defaults = objectValue(template.default_variables);
  const tag = config.tag || defaults.tag;
  return typeof tag === "string" && tag.trim() ? tag.trim() : "";
}

function slugAlias(value: string) {
  const slug = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
  if (/^[0-9]/.test(slug)) return `t_${slug}`;
  return slug || "template";
}

function defaultAlias(template: BindXrayTemplate) {
  return templateTag(template) || slugAlias(template.name || template.type);
}

function uniqueAlias(
  template: BindXrayTemplate,
  bindings: BindingRow[],
  alias = defaultAlias(template)
) {
  const used = new Set(
    bindings
      .filter((item) => item.template_id !== template.id)
      .map((item) => item.alias)
  );
  let next = alias;
  let index = 2;
  while (used.has(next)) {
    next = `${alias}_${index}`;
    index += 1;
  }
  return next;
}

function createBinding(
  template: BindXrayTemplate,
  bindings: BindingRow[],
  patch: Partial<BindingRow> = {}
): BindingRow {
  return {
    template_id: template.id,
    sort: bindings.length + 1,
    enabled: true,
    alias: uniqueAlias(template, bindings),
    variables_json: "{}",
    subscription_enabled: true,
    subscription_name: "",
    subscription_variables_json: "{}",
    ...patch,
  };
}

function serverBindingToRow(
  item: API.ServerXrayTemplate,
  templateById?: Map<number, BindXrayTemplate>
): BindingRow {
  const template = item.template || templateById?.get(item.template_id);
  return {
    template_id: item.template_id,
    sort: item.sort,
    enabled: item.enabled,
    alias: item.alias || template?.name || "",
    variables_json: formatJson(item.variables || {}),
    subscription_enabled: item.subscription_enabled ?? true,
    subscription_name: item.subscription_name || "",
    subscription_variables_json: formatJson(item.subscription_variables || {}),
  };
}

function bindingRowsSignature(rows: BindingRow[]) {
  return stableJsonKey(
    rows
      .slice()
      .sort((left, right) => {
        if (left.sort !== right.sort) return left.sort - right.sort;
        return left.template_id - right.template_id;
      })
      .map((item) => ({
        template_id: item.template_id,
        sort: Number(item.sort || 0),
        enabled: item.enabled !== false,
        alias: item.alias.trim(),
        variables: parseJsonObjectText(item.variables_json || "{}"),
        subscription_enabled: item.subscription_enabled !== false,
        subscription_name: item.subscription_name.trim(),
        subscription_variables: parseJsonObjectText(
          item.subscription_variables_json || "{}"
        ),
      }))
  );
}

function shortHash(value?: string) {
  if (!value) return "";
  return value.length > 10 ? value.slice(0, 10) : value;
}

function isServerOnline(status?: Partial<API.ServerStatus>) {
  return Boolean(status?.status && status.status !== "offline");
}

function isFailedStatus(value?: string) {
  return Boolean(value && value !== "ok" && value !== "pending");
}

function deriveConfigStatus({
  desiredError,
  desiredHash,
  hasDraftChanges,
  status,
}: {
  desiredError?: string;
  desiredHash?: string;
  hasDraftChanges: boolean;
  status?: Partial<API.ServerStatus>;
}): BindingStatusView {
  if (hasDraftChanges) {
    return {
      key: "draft",
      label: "有修改",
      description: "当前抽屉内有未保存的绑定修改，保存后才会下发到节点。",
    };
  }
  if (desiredError) {
    return {
      key: "invalid",
      label: "配置异常",
      description: desiredError,
    };
  }
  if (!isServerOnline(status)) {
    return {
      key: "offline",
      label: "节点离线",
      description: "节点未在线上报，暂时无法确认配置是否已经应用。",
    };
  }

  const applyStatus = status?.config_apply_status || "";
  const syncStatus = status?.config_sync_status || "";
  const error = status?.last_apply_error || status?.last_config_error || "";
  if (error || isFailedStatus(applyStatus) || isFailedStatus(syncStatus)) {
    return {
      key: "failed",
      label: "应用失败",
      description: error || applyStatus || syncStatus || "节点配置应用失败。",
    };
  }
  if (applyStatus === "pending") {
    return {
      key: "applying",
      label: "应用中",
      description: "节点已拉取配置，正在校验或重载 Xray。",
    };
  }
  if (
    desiredHash &&
    status?.running_config_hash === desiredHash &&
    applyStatus === "ok" &&
    syncStatus === "ok"
  ) {
    return {
      key: "effective",
      label: "已生效",
      description: "节点运行中的 Xray 配置与当前绑定配置一致。",
    };
  }
  if (desiredHash && status?.running_config_hash !== desiredHash) {
    return {
      key: "pending",
      label: "待生效",
      description: "绑定已保存，等待节点下一次拉取并应用配置。",
    };
  }
  return {
    key: "unknown",
    label: "待确认",
    description: "暂时无法根据节点上报判断配置状态。",
  };
}

function deriveBindingStatus(
  template: BindXrayTemplate,
  binding: BindingRow,
  configStatus: BindingStatusView
): BindingStatusView {
  if (binding.enabled === false || template.enabled === false) {
    return {
      key: "disabled",
      label: "未启用",
      description: "该绑定或模板已停用，不会进入节点运行配置。",
    };
  }
  return configStatus;
}

function BindingStatusBadge({
  compact,
  status,
}: {
  compact?: boolean;
  status: BindingStatusView;
}) {
  const style = BINDING_STATUS_STYLES[status.key];
  return (
    <Badge
      className={cn("gap-1.5 whitespace-nowrap border", style.badge)}
      variant="outline"
    >
      <span className={cn("h-1.5 w-1.5 rounded-full", style.dot)} />
      {compact ? status.label : status.label}
    </Badge>
  );
}

function collectRuleOutboundTags(value: unknown, result = new Set<string>()) {
  if (!Array.isArray(value)) return result;
  for (const item of value) {
    if (!isJsonObject(item)) continue;
    const rule = item;
    if (typeof rule.outboundTag === "string" && rule.outboundTag) {
      result.add(rule.outboundTag);
    }
    collectRuleOutboundTags(rule.rules, result);
  }
  return result;
}

function collectRuleBalancerTags(value: unknown, result = new Set<string>()) {
  if (!Array.isArray(value)) return result;
  for (const item of value) {
    if (!isJsonObject(item)) continue;
    const rule = item;
    if (typeof rule.balancerTag === "string" && rule.balancerTag) {
      result.add(rule.balancerTag);
    }
    collectRuleBalancerTags(rule.rules, result);
  }
  return result;
}

function collectBalancerTags(value: unknown) {
  const result = new Set<string>();
  if (!Array.isArray(value)) return result;
  for (const item of value) {
    if (!isJsonObject(item)) continue;
    const tag = item.tag;
    if (typeof tag === "string" && tag) {
      result.add(tag);
    }
  }
  return result;
}

function collectRoutingOutboundTags(template: BindXrayTemplate) {
  if (template.type !== "routing") return new Set<string>();
  const config = objectValue(template.config);
  return collectRuleOutboundTags(config.rules);
}

function isTemplateSearchMatch(template: BindXrayTemplate, keyword: string) {
  if (!keyword) return true;
  const haystack = [
    template.name,
    template.description,
    template.type,
    templateTag(template),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  return haystack.includes(keyword.toLowerCase());
}

function collectPreviewIssues(preview: ReturnType<typeof groupPreview>) {
  const issues: PreviewIssue[] = [];
  const outboundTags = new Set(
    (preview.outbounds || [])
      .map((item) => (isJsonObject(item) ? item.tag : undefined))
      .filter((tag): tag is string => typeof tag === "string" && !!tag)
  );
  const routing = isJsonObject(preview.routing) ? preview.routing : undefined;
  const balancerTags = collectBalancerTags(routing?.balancers);
  for (const tag of collectRuleOutboundTags(routing?.rules)) {
    if (!outboundTags.has(tag)) {
      issues.push({
        message: `路由引用的 outboundTag "${tag}" 尚未绑定对应出站模板。`,
      });
    }
  }
  for (const tag of collectRuleBalancerTags(routing?.rules)) {
    if (!balancerTags.has(tag)) {
      issues.push({
        message: `路由引用的 balancerTag "${tag}" 尚未定义对应负载均衡器。`,
      });
    }
  }
  return issues;
}

function collectBindingIssues(
  bindings: BindingRow[],
  templateById: Map<number, BindXrayTemplate>
) {
  const issues: PreviewIssue[] = [];
  const seen = new Map<string, string>();
  for (const binding of bindings.filter((item) => item.enabled !== false)) {
    const template = templateById.get(binding.template_id);
    if (!template) continue;
    const alias = binding.alias.trim();
    if (!alias) {
      issues.push({ message: `${template.name} 缺少别名。` });
      continue;
    }
    const variablesIssue = validateBindingJsonObject(
      binding.variables_json,
      `${template.name} 变量 JSON`
    );
    if (variablesIssue) {
      issues.push({ message: variablesIssue });
      continue;
    }
    const subscriptionVariablesIssue = validateBindingJsonObject(
      binding.subscription_variables_json,
      `${template.name} 订阅变量`
    );
    if (subscriptionVariablesIssue) {
      issues.push({ message: subscriptionVariablesIssue });
      continue;
    }
    const key = `${template.type}.${alias}`;
    const existing = seen.get(key);
    if (existing) {
      issues.push({
        message: `${template.type} 模板别名 "${alias}" 重复：${existing} / ${template.name}。`,
      });
      continue;
    }
    seen.set(key, template.name);
  }
  return issues;
}

function validateBindingJsonObject(value: string, label: string) {
  const text = value.trim();
  if (!text) return "";
  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return `${label} 必须是 JSON 对象。`;
    }
  } catch {
    return `${label} 不是合法 JSON。`;
  }
  return "";
}

function mergeGeodataObjects(items: JsonObject[]) {
  if (!items.length) return;
  if (items.length === 1) return items[0];
  const merged: JsonObject = {};
  for (const item of items) {
    Object.assign(merged, item);
    merged.assets = [...toArray(merged.assets), ...toArray(item.assets)];
  }
  return merged;
}

function cleanXrayConfig(source?: {
  inbounds?: unknown[];
  outbounds?: unknown[];
  dns?: unknown;
  routing?: unknown;
  geodata?: unknown;
}) {
  const config: JsonObject = {
    inbounds: source?.inbounds || [],
    outbounds: source?.outbounds || [],
  };
  if (source?.dns) {
    config.dns = source.dns;
  }
  if (source?.routing) {
    config.routing = source.routing;
  }
  if (source?.geodata) {
    config.geodata = source.geodata;
  }
  return config;
}

function variablePreview(source?: API.PreviewServerXrayConfigResponse) {
  return source?.variables || {};
}

function groupPreview(
  bindings: BindingRow[],
  templateById: Map<number, BindXrayTemplate>
) {
  const selected = selectEnabledTemplateBindings(bindings, templateById);

  const ref: Record<XrayTemplateType, Record<string, JsonObject>> = {
    inbound: {},
    outbound: {},
    dns: {},
    routing: {},
    geodata: {},
  };
  const rendered: RenderedTemplate[] = [];

  for (const { binding, template } of selected) {
    const variables = {
      ...objectValue(template.default_variables),
      ...extractTemplateConfigVariables({
        type: template.type as XrayTemplateType,
        config: objectValue(template.config),
      }),
      ...safeJsonParse(binding.variables_json || "", {}),
    };
    const source = template.config_template?.trim()
      ? template.config_template
      : JSON.stringify(template.config || {}, null, 2);
    const raw = renderTemplateString(source, {
      vars: variables,
      variables,
      ref,
      server: {},
      template,
      binding,
    });
    const item: RenderedTemplate = {
      id: template.id,
      name: template.name,
      type: template.type,
      alias: binding.alias,
      sort: binding.sort,
      variables,
      raw,
    };
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!isJsonObject(parsed)) {
        throw new Error("template rendered JSON must be an object");
      }
      const config = parsed;
      item.config = config;
      if (binding.alias) {
        ref[template.type] ||= {};
        const typeRef = ref[template.type];
        if (typeRef) {
          typeRef[binding.alias] = config;
        }
      }
    } catch (error) {
      item.error = error instanceof Error ? error.message : "JSON parse error";
    }
    rendered.push(item);
  }

  const renderedGroups = groupRenderedTemplates(rendered);

  return {
    note: "前端草稿预览：会合并默认变量和绑定变量，并解析常见的 {{ .Vars.xxx }}、{{ toJson .Vars.xxx }}、{{ .Ref.type.alias.field }}。最终配置仍以后端节点拉取渲染结果为准。",
    inbounds: renderedGroups.inbounds,
    outbounds: renderedGroups.outbounds,
    dns: mergeDnsObjects(renderedGroups.dnsItems),
    routing: mergeRoutingObjects(renderedGroups.routingItems),
    geodata: mergeGeodataObjects(renderedGroups.geodataItems),
    rendered,
    errors: renderedGroups.errors,
  };
}

export default function ServerXrayTemplateBindForm({
  onOpenChange,
  open: controlledOpen,
  server,
  trigger,
}: {
  onOpenChange?: (open: boolean) => void;
  open?: boolean;
  server: API.Server;
  trigger?: React.ReactNode;
}) {
  const { t } = useTranslation("xray-templates");
  const [innerOpen, setInnerOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [bindings, setBindings] = useState<BindingRow[]>([]);
  const [activeTab, setActiveTab] = useState<BindTab>("inbound");
  const [templateSearch, setTemplateSearch] = useState("");
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);
  const hasManualDraftRef = useRef(false);
  const open = controlledOpen ?? innerOpen;
  const setOpen = (value: boolean) => {
    onOpenChange?.(value);
    if (controlledOpen === undefined) {
      setInnerOpen(value);
    }
  };

  const { data: templateData } = useQuery({
    enabled: open,
    queryKey: ["xray-template-list-all"],
    queryFn: async () => {
      const { data } = await queryXrayTemplateCatalog({
        page: 1,
        size: 1000,
      });
      return normalizeListResult(data.data);
    },
  });
  const templates = useMemo(() => templateData || [], [templateData]);
  const templateById = useMemo(
    () => new Map(templates.map((template) => [template.id, template])),
    [templates]
  );
  const outboundTemplateByTag = useMemo(() => {
    const result = new Map<string, BindXrayTemplate>();
    for (const template of templates) {
      if (template.type === "outbound") {
        result.set(templateTag(template), template);
      }
    }
    return result;
  }, [templates]);

  const { data: serverBindingData, refetch } = useQuery({
    enabled: open,
    queryKey: ["server-xray-template-list", server.id],
    queryFn: async () => {
      const { data } = await queryServerXrayTemplateList({
        server_id: server.id,
        include_status: true,
        include_desired_config: true,
      });
      return data.data;
    },
    refetchInterval: open ? 5000 : false,
  });
  const serverBindings = useMemo(
    () => normalizeListResult(serverBindingData),
    [serverBindingData]
  );
  const savedBindingRows = useMemo(
    () => serverBindings.map((item) => serverBindingToRow(item, templateById)),
    [serverBindings, templateById]
  );
  const serverStatus = serverBindingData?.status || server.status;
  const desiredConfigHash = serverBindingData?.desired_config_hash || "";
  const desiredConfigError = serverBindingData?.desired_config_error || "";
  const savedBindingSignature = useMemo(
    () => bindingRowsSignature(savedBindingRows),
    [savedBindingRows]
  );
  const draftBindingSignature = useMemo(
    () => bindingRowsSignature(bindings),
    [bindings]
  );

  useEffect(() => {
    if (!open) {
      hasManualDraftRef.current = false;
      return;
    }
    if (!serverBindingData) return;
    if (hasManualDraftRef.current) return;
    setBindings(savedBindingRows);
  }, [open, savedBindingRows, serverBindingData]);

  const hasDraftChanges = useMemo(
    () =>
      Boolean(serverBindingData) &&
      open &&
      draftBindingSignature !== savedBindingSignature,
    [draftBindingSignature, open, savedBindingSignature, serverBindingData]
  );
  const configStatus = useMemo(
    () =>
      deriveConfigStatus({
        desiredError: desiredConfigError,
        desiredHash: desiredConfigHash,
        hasDraftChanges,
        status: serverStatus,
      }),
    [desiredConfigError, desiredConfigHash, hasDraftChanges, serverStatus]
  );

  const preview = useMemo(
    () => groupPreview(bindings, templateById),
    [bindings, templateById]
  );
  const previewIssues = useMemo(
    () => [
      ...collectBindingIssues(bindings, templateById),
      ...collectPreviewIssues(preview),
    ],
    [bindings, preview, templateById]
  );
  const selectedBindings = useMemo(
    () =>
      bindings
        .slice()
        .sort((a, b) => a.sort - b.sort)
        .map((binding) => ({
          binding,
          template: templateById.get(binding.template_id),
        }))
        .filter((item) => item.template) as {
        binding: BindingRow;
        template: BindXrayTemplate;
      }[],
    [bindings, templateById]
  );
  const selectedByType = useMemo(() => {
    const result = {} as Record<XrayTemplateType, number>;
    for (const type of TYPE_ORDER) {
      result[type] = 0;
    }
    for (const item of selectedBindings) {
      result[item.template.type] += 1;
    }
    return result;
  }, [selectedBindings]);
  const templatesByType = useMemo(() => {
    const result = {} as Record<XrayTemplateType, number>;
    for (const type of TYPE_ORDER) {
      result[type] = 0;
    }
    for (const template of templates) {
      result[template.type] += 1;
    }
    return result;
  }, [templates]);

  const previewPayload = useMemo<API.PreviewServerXrayConfigRequest>(
    () => ({
      server_id: server.id,
      bindings: bindings
        .filter((item) => item.enabled)
        .slice()
        .sort((a, b) => a.sort - b.sort)
        .map((item) => ({
          template_id: item.template_id,
          sort: item.sort,
          enabled: item.enabled,
          alias: item.alias,
          variables: safeJsonParse(item.variables_json || "", {}),
          subscription_enabled: item.subscription_enabled,
          subscription_name: item.subscription_name,
          subscription_variables: safeJsonParse(
            item.subscription_variables_json || "",
            {}
          ),
        })),
    }),
    [bindings, server.id]
  );
  const bindingByTemplateId = useMemo(
    () => buildBindingByTemplateId(bindings),
    [bindings]
  );

  const {
    data: backendPreview,
    error: backendPreviewError,
    isFetching: previewFetching,
  } = useQuery({
    enabled: open && previewIssues.length === 0,
    queryKey: ["server-xray-template-preview", previewPayload],
    queryFn: async () => {
      const { data } = await previewServerXrayConfig(previewPayload);
      return data.data;
    },
    retry: false,
  });

  function getBinding(templateId: number) {
    return bindingByTemplateId.get(templateId);
  }

  function updateBinding(
    template: BindXrayTemplate,
    patch: Partial<BindingRow>
  ) {
    hasManualDraftRef.current = true;
    setBindings((current) => {
      const index = findBindingIndexByTemplateId(current, template.id);
      if (index < 0) {
        return [...current, createBinding(template, current, patch)];
      }
      const existing = current[index];
      if (!existing) {
        return current;
      }
      const next = current.slice();
      next[index] = { ...existing, ...patch };
      return next;
    });
  }

  function enableBinding(template: BindXrayTemplate) {
    hasManualDraftRef.current = true;
    setBindings((current) => {
      const index = findBindingIndexByTemplateId(current, template.id);
      if (index >= 0) {
        const existing = current[index];
        if (!existing) {
          return current;
        }
        const next = current.slice();
        next[index] = { ...existing, enabled: true };
        return next;
      }
      const next = [...current];
      const requiredOutboundTags = collectRoutingOutboundTags(template);
      for (const tag of requiredOutboundTags) {
        const hasOutbound = next.some((binding) => {
          const boundTemplate = templateById.get(binding.template_id);
          return (
            boundTemplate?.type === "outbound" &&
            binding.enabled !== false &&
            (templateTag(boundTemplate) || binding.alias) === tag
          );
        });
        if (hasOutbound) continue;
        const dependency = outboundTemplateByTag.get(tag);
        if (dependency) {
          next.push(createBinding(dependency, next, { enabled: true }));
        }
      }
      next.push(createBinding(template, next, { enabled: true }));
      return next.map((item, index) => ({
        ...item,
        sort: item.sort || index + 1,
      }));
    });
  }

  function removeBinding(templateId: number) {
    hasManualDraftRef.current = true;
    setBindings((current) =>
      current.filter((item) => item.template_id !== templateId)
    );
  }

  async function save() {
    if (previewIssues.length) {
      toast.error(previewIssues[0]?.message || "Xray 模板绑定配置不完整");
      return;
    }
    if (backendPreviewError) {
      toast.error(
        backendPreviewError instanceof Error
          ? backendPreviewError.message
          : "Xray 模板预览校验失败"
      );
      return;
    }
    setSaving(true);
    try {
      await bindServerXrayTemplates({
        server_id: server.id,
        bindings: bindings
          .slice()
          .sort((a, b) => a.sort - b.sort)
          .map((item) => ({
            template_id: item.template_id,
            sort: item.sort,
            enabled: item.enabled,
            alias: item.alias,
            variables: safeJsonParse(item.variables_json || "", {}),
            subscription_enabled: item.subscription_enabled,
            subscription_name: item.subscription_name,
            subscription_variables: safeJsonParse(
              item.subscription_variables_json || "",
              {}
            ),
          })),
      });
      toast.success(t("message.bound", "Saved"));
      hasManualDraftRef.current = false;
      const refreshed = await refetch();
      setBindings(
        normalizeListResult(refreshed.data).map((item) =>
          serverBindingToRow(item, templateById)
        )
      );
    } finally {
      setSaving(false);
    }
  }

  function updateBindingVariables(
    template: BindXrayTemplate,
    binding: BindingRow | undefined,
    patch: JsonObject
  ) {
    const current = parseJsonObjectText(binding?.variables_json || "{}");
    updateBinding(template, {
      variables_json: formatJson({ ...current, ...patch }),
    });
  }

  function removeBindingVariable(
    template: BindXrayTemplate,
    binding: BindingRow | undefined,
    key: string
  ) {
    const current = parseJsonObjectText(binding?.variables_json || "{}");
    delete current[key];
    updateBinding(template, { variables_json: formatJson(current) });
  }

  function renderBindingVariablesEditor(
    template: BindXrayTemplate,
    binding: BindingRow | undefined
  ) {
    const hints = collectVariableHints(template);
    const bindingVariables = parseJsonObjectText(
      binding?.variables_json || "{}"
    );
    const configVariables = extractTemplateConfigVariables({
      type: template.type as XrayTemplateType,
      config: objectValue(template.config),
    });
    const defaultVariables = objectValue(template.default_variables);
    const inherited = inheritedVariables(template);

    function sourceMeta(key: string) {
      if (hasOwnValue(bindingVariables, key)) {
        return {
          label: "服务器覆盖",
          className: VARIABLE_SOURCE_STYLES.override,
        };
      }
      if (hasOwnValue(configVariables, key)) {
        return {
          label: "模板配置",
          className: VARIABLE_SOURCE_STYLES.config,
        };
      }
      if (hasOwnValue(defaultVariables, key)) {
        return {
          label: "模板默认",
          className: VARIABLE_SOURCE_STYLES.default,
        };
      }
      return {
        label: "未设置",
        className: VARIABLE_SOURCE_STYLES.empty,
      };
    }

    function SourceBadge({ fieldKey }: { fieldKey: string }) {
      const source = sourceMeta(fieldKey);
      return (
        <Badge className={cn("border px-1.5 py-0", source.className)}>
          {source.label}
        </Badge>
      );
    }

    if (!hints.length) {
      return (
        <JsonObjectEditor
          addLabel="添加节点变量"
          emptyText="该模板没有定义变量结构，可按需新增键值。"
          onChange={(value) =>
            updateBinding(template, { variables_json: value || "{}" })
          }
          value={binding?.variables_json || "{}"}
        />
      );
    }

    const hintKeys = new Set(hints.map((hint) => hint.key));
    const knownValues = Object.fromEntries(
      Object.entries(bindingVariables).filter(([key]) => hintKeys.has(key))
    );
    const extraValues = Object.fromEntries(
      Object.entries(bindingVariables).filter(([key]) => !hintKeys.has(key))
    );

    function commitExtraVariables(value: string) {
      updateBinding(template, {
        variables_json: formatJson({
          ...knownValues,
          ...parseJsonObjectText(value),
        }),
      });
    }

    return (
      <div className="space-y-3">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {hints.map((hint) => {
            const hasOverride = hasOwnValue(bindingVariables, hint.key);
            const currentValue = bindingVariables[hint.key];
            const inheritedValue = inherited[hint.key] ?? hint.defaultValue;
            const effectiveValue = hasOverride ? currentValue : inheritedValue;
            const docHelp = getXrayFieldHelp(hint.key);
            const helperText = hint.description || docHelp?.hint || hint.key;
            const value =
              !hasOverride ||
              currentValue === undefined ||
              currentValue === null
                ? ""
                : String(currentValue);
            const placeholder = stringifyHintValue(inheritedValue);

            if (hint.type === "boolean") {
              return (
                <div
                  className="rounded-md border bg-background px-3 py-2"
                  key={hint.key}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 font-medium text-sm">
                        <span>
                          {hint.title || hint.key}
                          {hint.required ? (
                            <span className="text-destructive"> *</span>
                          ) : null}
                        </span>
                        <SourceBadge fieldKey={hint.key} />
                        <XrayFieldHelp fieldKey={hint.key} />
                      </div>
                      <div className="truncate text-muted-foreground text-xs">
                        {helperText}
                      </div>
                    </div>
                    <Switch
                      checked={!!effectiveValue}
                      onCheckedChange={(checked) =>
                        updateBindingVariables(template, binding, {
                          [hint.key]: checked,
                        })
                      }
                    />
                  </div>
                  <div className="mt-2 flex items-center justify-between gap-2 text-muted-foreground text-xs">
                    <span>生效值：{effectiveValue ? "开启" : "关闭"}</span>
                    {hasOverride ? (
                      <Button
                        className="h-6 px-2 text-xs"
                        onClick={() =>
                          removeBindingVariable(template, binding, hint.key)
                        }
                        type="button"
                        variant="ghost"
                      >
                        继承模板
                      </Button>
                    ) : null}
                  </div>
                </div>
              );
            }

            if (hint.type === "array" || hint.type === "object") {
              return (
                <div className="space-y-1.5" key={hint.key}>
                  <div className="flex items-center gap-2 text-muted-foreground text-xs">
                    <span>
                      {hint.title || hint.key}
                      {hint.required ? (
                        <span className="text-destructive"> *</span>
                      ) : null}
                    </span>
                    <SourceBadge fieldKey={hint.key} />
                    <XrayFieldHelp fieldKey={hint.key} />
                  </div>
                  <Textarea
                    className="min-h-20 font-mono text-xs"
                    onChange={(event) => {
                      const text = event.target.value.trim();
                      if (!text) {
                        removeBindingVariable(template, binding, hint.key);
                        return;
                      }
                      try {
                        updateBindingVariables(template, binding, {
                          [hint.key]: JSON.parse(text),
                        });
                      } catch {
                        updateBindingVariables(template, binding, {
                          [hint.key]: text,
                        });
                      }
                    }}
                    placeholder={placeholder || "继承模板 JSON"}
                    value={
                      !hasOverride || currentValue === undefined
                        ? ""
                        : JSON.stringify(currentValue, null, 2)
                    }
                  />
                  <div className="flex items-center justify-between gap-2 text-muted-foreground text-xs">
                    <span className="truncate">
                      {helperText !== hint.key ? `${helperText} · ` : ""}
                      生效值：{compactValue(effectiveValue)}
                    </span>
                    {hasOverride ? (
                      <Button
                        className="h-6 px-2 text-xs"
                        onClick={() =>
                          removeBindingVariable(template, binding, hint.key)
                        }
                        type="button"
                        variant="ghost"
                      >
                        继承模板
                      </Button>
                    ) : null}
                  </div>
                </div>
              );
            }

            return (
              <div className="space-y-1.5" key={hint.key}>
                <div className="flex items-center gap-2 text-muted-foreground text-xs">
                  <span>
                    {hint.title || hint.key}
                    {hint.required ? (
                      <span className="text-destructive"> *</span>
                    ) : null}
                  </span>
                  <SourceBadge fieldKey={hint.key} />
                  <XrayFieldHelp fieldKey={hint.key} />
                </div>
                <EnhancedInput
                  onValueChange={(nextValue) => {
                    if (!nextValue) {
                      removeBindingVariable(template, binding, hint.key);
                      return;
                    }
                    updateBindingVariables(template, binding, {
                      [hint.key]:
                        hint.type === "number" ? Number(nextValue) : nextValue,
                    });
                  }}
                  placeholder={placeholder}
                  type={hint.type === "number" ? "number" : "text"}
                  value={value}
                />
                <div className="flex items-center justify-between gap-2 text-muted-foreground text-xs">
                  <span className="truncate">
                    {helperText !== hint.key ? `${helperText} · ` : ""}
                    生效值：{compactValue(effectiveValue)}
                  </span>
                  {hasOverride ? (
                    <Button
                      className="h-6 px-2 text-xs"
                      onClick={() =>
                        removeBindingVariable(template, binding, hint.key)
                      }
                      type="button"
                      variant="ghost"
                    >
                      继承模板
                    </Button>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
        <div className="space-y-1.5">
          <div className="text-muted-foreground text-xs">其他变量</div>
          <JsonObjectEditor
            addLabel="添加其他变量"
            emptyText="没有额外变量。"
            onChange={commitExtraVariables}
            value={formatJson(extraValues)}
          />
        </div>
      </div>
    );
  }

  function renderSubscriptionVariablesEditor(
    template: BindXrayTemplate,
    binding: BindingRow | undefined
  ) {
    const meta = objectValue(template.subscription_meta);
    const metaKeys = Object.keys(meta);
    const values = parseJsonObjectText(
      binding?.subscription_variables_json || "{}"
    );
    if (!metaKeys.length) {
      return (
        <JsonObjectEditor
          addLabel="添加订阅变量"
          emptyText="该模板没有订阅元数据，可按需新增变量。"
          onChange={(value) =>
            updateBinding(template, {
              subscription_variables_json: value || "{}",
            })
          }
          value={binding?.subscription_variables_json || "{}"}
        />
      );
    }

    const metaKeySet = new Set(metaKeys);
    const knownValues = Object.fromEntries(
      Object.entries(values).filter(([key]) => metaKeySet.has(key))
    );
    const extraValues = Object.fromEntries(
      Object.entries(values).filter(([key]) => !metaKeySet.has(key))
    );

    function commitKnown(key: string, value: string) {
      const next = { ...values };
      if (value) {
        next[key] = value;
      } else {
        delete next[key];
      }
      updateBinding(template, {
        subscription_variables_json: formatJson(next),
      });
    }

    function commitExtra(value: string) {
      updateBinding(template, {
        subscription_variables_json: formatJson({
          ...knownValues,
          ...parseJsonObjectText(value),
        }),
      });
    }

    return (
      <div className="space-y-3">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {metaKeys.map((key) => (
            <div className="space-y-1.5" key={key}>
              <div className="text-muted-foreground text-xs">{key}</div>
              <EnhancedInput
                onValueChange={(value) => commitKnown(key, value)}
                placeholder={String(meta[key] ?? "")}
                value={String(values[key] ?? "")}
              />
            </div>
          ))}
        </div>
        <div className="space-y-1.5">
          <div className="text-muted-foreground text-xs">其他订阅变量</div>
          <JsonObjectEditor
            addLabel="添加订阅变量"
            emptyText="没有额外订阅变量。"
            onChange={commitExtra}
            value={formatJson(extraValues)}
          />
        </div>
      </div>
    );
  }

  function renderSelectedSummary() {
    if (!selectedBindings.length) {
      return (
        <Empty className="border p-4">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Braces />
            </EmptyMedia>
            <EmptyDescription>
              还没有选择模板。先从右侧选择 inbound，再补齐 outbound、DNS
              和路由。
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      );
    }
    return (
      <div className="space-y-2">
        {selectedBindings.map(({ binding, template }) => {
          const status = deriveBindingStatus(template, binding, configStatus);
          return (
            <div
              className={cn(
                "grid grid-cols-[1fr_auto] items-center gap-2 rounded-md border bg-background px-3 py-2",
                BINDING_STATUS_STYLES[status.key].card
              )}
              key={binding.template_id}
            >
              <div className="min-w-0">
                <div className="flex min-w-0 items-center gap-2">
                  <Badge
                    className={TYPE_ACCENTS[template.type]}
                    variant="outline"
                  >
                    {t(`type.${template.type}`, TYPE_LABELS[template.type])}
                  </Badge>
                  <span className="truncate font-medium text-sm">
                    {template.name}
                  </span>
                  <BindingStatusBadge compact status={status} />
                </div>
                <div className="truncate text-muted-foreground text-xs">
                  {binding.alias} · #{binding.sort || 0}
                </div>
              </div>
              <Button
                onClick={() => removeBinding(template.id)}
                size="sm"
                type="button"
                variant="ghost"
              >
                {t("action.remove", "移除")}
              </Button>
            </div>
          );
        })}
      </div>
    );
  }

  function renderTemplateList(type: XrayTemplateType) {
    const list = templates
      .filter((item) => item.type === type)
      .filter((item) => isTemplateSearchMatch(item, templateSearch))
      .filter((item) => !showSelectedOnly || Boolean(getBinding(item.id)))
      .sort((left, right) => {
        const leftSelected = getBinding(left.id) ? 0 : 1;
        const rightSelected = getBinding(right.id) ? 0 : 1;
        if (leftSelected !== rightSelected) return leftSelected - rightSelected;
        return left.name.localeCompare(right.name);
      });
    const typeLabel = t(`type.${type}`, TYPE_LABELS[type]);
    const typeCount = templatesByType[type] || 0;
    const selectedCount = selectedByType[type] || 0;

    const toolbar = (
      <div className="grid grid-cols-1 gap-2 md:grid-cols-[1fr_auto_auto]">
        <EnhancedInput
          onValueChange={setTemplateSearch}
          placeholder={`搜索 ${typeLabel}`}
          value={templateSearch}
        />
        <Badge className="justify-center px-3" variant="outline">
          {selectedCount}/{typeCount}
        </Badge>
        <Button
          onClick={() => setShowSelectedOnly((value) => !value)}
          type="button"
          variant={showSelectedOnly ? "default" : "outline"}
        >
          {showSelectedOnly ? "全部模板" : "仅看已选"}
        </Button>
      </div>
    );

    if (!list.length) {
      return (
        <div className="space-y-3">
          {toolbar}
          <Empty className="border p-8">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Braces />
              </EmptyMedia>
              <EmptyDescription>没有匹配的 {typeLabel} 模板。</EmptyDescription>
            </EmptyHeader>
          </Empty>
        </div>
      );
    }

    return (
      <div className="space-y-3">
        {toolbar}
        {list.map((template) => {
          const binding = getBinding(template.id);
          const selected = Boolean(binding);
          const bindingStatus = binding
            ? deriveBindingStatus(template, binding, configStatus)
            : undefined;
          const requiredOutboundTags = collectRoutingOutboundTags(template);
          return (
            <div
              className={cn(
                "space-y-3 rounded-md border bg-background p-3 transition-colors",
                selected && bindingStatus
                  ? BINDING_STATUS_STYLES[bindingStatus.key].card
                  : ""
              )}
              key={template.id}
            >
              <div className="grid grid-cols-[1fr_auto_auto] items-center gap-3">
                <div className="min-w-0">
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="truncate font-medium">
                      {template.name}
                    </span>
                    <Badge
                      className={TYPE_ACCENTS[template.type]}
                      variant="outline"
                    >
                      {typeLabel}
                    </Badge>
                    <Badge variant={template.enabled ? "secondary" : "outline"}>
                      {template.enabled ? "启用" : "停用"}
                    </Badge>
                    {bindingStatus ? (
                      <BindingStatusBadge compact status={bindingStatus} />
                    ) : null}
                  </div>
                  <p className="truncate text-muted-foreground text-xs">
                    {templateDescription(template)}
                  </p>
                  {requiredOutboundTags.size ? (
                    <p className="text-muted-foreground text-xs">
                      依赖出站：{[...requiredOutboundTags].join(", ")}
                    </p>
                  ) : null}
                </div>
                <div className="w-20 space-y-1">
                  <EnhancedInput
                    disabled={!selected}
                    min={0}
                    onValueChange={(value) =>
                      updateBinding(template, { sort: Number(value) || 0 })
                    }
                    placeholder="排序"
                    step={1}
                    type="number"
                    value={binding?.sort ?? 0}
                  />
                  <p className="text-center text-muted-foreground text-xs">
                    顺序
                  </p>
                </div>
                <Switch
                  checked={selected && binding?.enabled !== false}
                  onCheckedChange={(checked) => {
                    if (!checked) {
                      removeBinding(template.id);
                      return;
                    }
                    enableBinding(template);
                  }}
                />
              </div>
              {selected ? (
                <div className="space-y-4 rounded-md border bg-background/80 p-3">
                  <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                    <div className="space-y-1.5">
                      <div className="text-muted-foreground text-xs">
                        绑定别名
                      </div>
                      <EnhancedInput
                        onValueChange={(value) =>
                          updateBinding(template, { alias: value })
                        }
                        placeholder={t("bind.alias", "Alias")}
                        value={binding?.alias || ""}
                      />
                      <p className="text-muted-foreground text-xs">
                        引用示例：{" "}
                        <span className="font-mono">
                          {referenceExample(
                            template.type,
                            binding?.alias || ""
                          )}
                        </span>
                      </p>
                    </div>
                    <div className="space-y-1.5">
                      <div className="text-muted-foreground text-xs">
                        订阅展示名
                      </div>
                      <EnhancedInput
                        onValueChange={(value) =>
                          updateBinding(template, { subscription_name: value })
                        }
                        placeholder={t(
                          "bind.subscriptionName",
                          "Subscription name template"
                        )}
                        value={binding?.subscription_name || ""}
                      />
                      <p className="text-muted-foreground text-xs">
                        可选 Go 模板。留空时使用服务器名和绑定别名。
                      </p>
                    </div>
                  </div>
                  <div className="space-y-3">
                    <div className="space-y-2">
                      <div>
                        <div className="font-medium text-sm">节点变量</div>
                        <p className="text-muted-foreground text-xs">
                          覆盖端口、tag、SNI、path 等每台服务器不同的值。
                        </p>
                      </div>
                      {renderBindingVariablesEditor(template, binding)}
                    </div>
                    <div className="space-y-2 rounded-md border bg-muted/20 p-3">
                      <div className="flex items-center justify-between rounded-md border bg-background px-3 py-2">
                        <span className="text-sm">加入订阅</span>
                        <Switch
                          checked={binding?.subscription_enabled !== false}
                          onCheckedChange={(checked) =>
                            updateBinding(template, {
                              subscription_enabled: checked,
                            })
                          }
                        />
                      </div>
                      {renderSubscriptionHints(template)}
                      {renderSubscriptionVariablesEditor(template, binding)}
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      {trigger ? <SheetTrigger asChild>{trigger}</SheetTrigger> : null}
      <SheetContent className="h-dvh w-[min(1180px,calc(100vw-24px))] max-w-full gap-0 overflow-hidden p-0 pb-[env(safe-area-inset-bottom)] md:max-w-6xl">
        <SheetHeader className="shrink-0 border-b px-6 py-4 pr-12">
          <SheetTitle className="flex flex-wrap items-center gap-2 break-words">
            {t("bind.title", "绑定 Xray 模板")} · {server.name}
            <Badge variant="secondary">{selectedBindings.length} 个模板</Badge>
            <Badge variant={previewIssues.length ? "outline" : "secondary"}>
              {previewIssues.length
                ? "需要处理"
                : backendPreviewError
                  ? "预览失败"
                  : "配置可预览"}
            </Badge>
            <BindingStatusBadge status={configStatus} />
          </SheetTitle>
          <SheetDescription className="sr-only">
            Bind Xray templates to this server, edit binding variables, and
            preview the rendered configuration.
          </SheetDescription>
        </SheetHeader>

        <div
          className="grid min-h-0 flex-1 auto-rows-max grid-cols-1 gap-4 overflow-y-auto overscroll-contain px-6 py-4 lg:grid-cols-[310px_minmax(0,1fr)] lg:grid-rows-[minmax(0,1fr)] lg:overflow-hidden"
          data-testid="xray-binding-scroll"
        >
          <aside className="min-w-0 space-y-4 rounded-md border bg-muted/20 p-3 lg:min-h-0 lg:overflow-y-auto lg:overscroll-contain">
            <div
              className={cn(
                "space-y-3 rounded-md border bg-background p-3",
                BINDING_STATUS_STYLES[configStatus.key].card
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="font-medium text-sm">节点配置状态</div>
                  <div className="line-clamp-2 text-muted-foreground text-xs">
                    {configStatus.description}
                  </div>
                </div>
                <BindingStatusBadge compact status={configStatus} />
              </div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="rounded-md border bg-background/70 px-2 py-1.5">
                  <div className="text-muted-foreground">期望</div>
                  <div className="font-medium font-mono">
                    {shortHash(desiredConfigHash) || "-"}
                  </div>
                </div>
                <div className="rounded-md border bg-background/70 px-2 py-1.5">
                  <div className="text-muted-foreground">运行</div>
                  <div className="font-medium font-mono">
                    {shortHash(
                      serverStatus?.running_config_hash ||
                        serverStatus?.config_version
                    ) || "-"}
                  </div>
                </div>
                {serverStatus?.pending_config_hash ? (
                  <div className="col-span-2 rounded-md border bg-background/70 px-2 py-1.5">
                    <div className="text-muted-foreground">待应用</div>
                    <div className="font-medium font-mono">
                      {shortHash(serverStatus.pending_config_hash)}
                    </div>
                  </div>
                ) : null}
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <div className="font-medium text-sm">
                    {t("bind.selectedTemplates", "已绑定流水线")}
                  </div>
                  <div className="text-muted-foreground text-xs">
                    按顺序渲染，路由可引用前面的模板。
                  </div>
                </div>
                <Badge variant="secondary">{selectedBindings.length}</Badge>
              </div>
              {renderSelectedSummary()}
            </div>

            <div className="space-y-2 rounded-md border bg-background p-3">
              <div className="font-medium text-sm">模板覆盖</div>
              <div className="grid grid-cols-2 gap-2">
                {TYPE_ORDER.map((item) => (
                  <div
                    className="rounded-md border bg-muted/20 px-2 py-1.5"
                    key={item}
                  >
                    <div className="text-muted-foreground text-xs">
                      {t(`type.${item}`, TYPE_LABELS[item])}
                    </div>
                    <div className="font-medium text-sm tabular-nums">
                      {selectedByType[item] || 0}/{templatesByType[item] || 0}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {previewIssues.length ? (
              <Alert>
                <TriangleAlert />
                <AlertTitle>本地校验提示</AlertTitle>
                <AlertDescription>{previewIssues[0]?.message}</AlertDescription>
              </Alert>
            ) : null}
            {!previewIssues.length && backendPreviewError ? (
              <Alert variant="destructive">
                <TriangleAlert />
                <AlertTitle>服务端校验失败</AlertTitle>
                <AlertDescription>
                  {backendPreviewError instanceof Error
                    ? backendPreviewError.message
                    : "Xray 模板预览校验失败"}
                </AlertDescription>
              </Alert>
            ) : null}
            {previewIssues.length || backendPreviewError ? null : (
              <Alert>
                <CircleCheck />
                <AlertTitle>本地校验通过</AlertTitle>
                <AlertDescription>
                  配置已通过本地校验，保存前会继续以服务端预览结果为准。
                </AlertDescription>
              </Alert>
            )}
          </aside>

          <div className="min-w-0 rounded-md border bg-background p-4 lg:min-h-0 lg:overflow-y-auto lg:overscroll-contain">
            <Tabs
              onValueChange={(value) => setActiveTab(value as BindTab)}
              value={activeTab}
            >
              <TabsList className="w-full justify-start overflow-x-auto">
                <TabsTrigger value="inbound">
                  {t("type.inbound", "Inbound")}
                </TabsTrigger>
                <TabsTrigger value="outbound">
                  {t("type.outbound", "Outbound")}
                </TabsTrigger>
                <TabsTrigger value="dns">{t("type.dns", "DNS")}</TabsTrigger>
                <TabsTrigger value="routing">
                  {t("type.routing", "Routing")}
                </TabsTrigger>
                <TabsTrigger value="geodata">
                  {t("type.geodata", "Geodata")}
                </TabsTrigger>
                <TabsTrigger value="preview">
                  {t("bind.preview", "Preview")}
                </TabsTrigger>
                <TabsTrigger value="variables">
                  {t("bind.variablesPreview", "Variables")}
                </TabsTrigger>
              </TabsList>
              <TabsContent className="pt-4" value="inbound">
                {renderTemplateList("inbound")}
              </TabsContent>
              <TabsContent className="pt-4" value="outbound">
                {renderTemplateList("outbound")}
              </TabsContent>
              <TabsContent className="pt-4" value="dns">
                {renderTemplateList("dns")}
              </TabsContent>
              <TabsContent className="pt-4" value="routing">
                {renderTemplateList("routing")}
              </TabsContent>
              <TabsContent className="pt-4" value="geodata">
                {renderTemplateList("geodata")}
              </TabsContent>
              <TabsContent className="pt-4" value="preview">
                <Textarea
                  className="min-h-[520px] font-mono text-xs"
                  readOnly
                  value={JSON.stringify(
                    cleanXrayConfig(backendPreview || preview),
                    null,
                    2
                  )}
                />
              </TabsContent>
              <TabsContent className="pt-4" value="variables">
                {previewFetching ? (
                  <div className="mb-2 text-muted-foreground text-xs">
                    {t("state.loading", "Loading")}...
                  </div>
                ) : null}
                <Textarea
                  className="min-h-[520px] font-mono text-xs"
                  readOnly
                  value={JSON.stringify(
                    variablePreview(backendPreview),
                    null,
                    2
                  )}
                />
              </TabsContent>
            </Tabs>
          </div>
        </div>

        <SheetFooter className="shrink-0 flex-row justify-end gap-2 border-t px-6 py-4">
          <Button
            onClick={() => setOpen(false)}
            type="button"
            variant="outline"
          >
            {t("action.cancel", "Cancel")}
          </Button>
          <Button disabled={saving || previewFetching} onClick={save}>
            {saving ? (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            ) : null}
            {t("action.save", "Save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
