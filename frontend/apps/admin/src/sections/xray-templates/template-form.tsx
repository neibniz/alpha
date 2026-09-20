import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
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
import {
  Braces,
  CircleAlert,
  FileCode2,
  Info,
  Share2,
  SlidersHorizontal,
} from "lucide-react";
import {
  cloneElement,
  isValidElement,
  type FormEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  type Control,
  type FieldErrors,
  type FieldPath,
  type Resolver,
  useForm,
  useWatch,
} from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { DiscardChangesDialog } from "../../components/discard-changes-dialog";
import {
  generateRealityKeyPair,
  generateRealityShortId,
} from "../servers/generate";
import {
  buildDnsConfig,
  buildGeodataConfig,
  buildInboundConfig,
  buildOutboundConfig,
  buildRoutingConfig,
  configToFormValues,
  extractTemplateConfigVariables,
  formatJson,
  INBOUND_PROTOCOLS,
  linesToArray,
  NETWORKS,
  OUTBOUND_PROTOCOLS,
  QUERY_STRATEGIES,
  ROUTING_DOMAIN_STRATEGIES,
  ROUTING_PRESETS,
  SECURITIES,
  safeJsonParse,
  normalizeVlessRealityGrpcConfig,
  XRAY_TEMPLATE_TYPES,
  type XrayTemplateType,
} from "./config";
import {
  type JsonArrayColumn,
  JsonArrayObjectEditor,
  JsonObjectEditor,
  parseJsonObjectText,
} from "./json-form-controls";
import {
  isFiveFieldCron,
  isRealityShortId,
  isRealityShortIdList,
  isRealityTarget,
  isValidPort,
  isVlessFlowAllowed,
  isXrayVersion,
  parseOptionalInteger,
} from "./template-form-validation";
import { XrayFieldDescription, XrayFieldLabel } from "./xray-field-help";

type JsonObject = Record<string, unknown>;
type XrayFormControl = Control<FormValues>;
type FormFieldName = FieldPath<FormValues>;
type FormRootFieldName = Extract<keyof FormValues, string>;
type XrayFormTab = "form" | "advanced" | "template" | "subscription";

function tabForField(name: string): XrayFormTab {
  if (name === "advanced_json") return "advanced";
  if (
    [
      "config_template",
      "variables_schema_json",
      "default_variables_json",
    ].includes(name)
  ) {
    return "template";
  }
  if (name === "subscription_meta_json") return "subscription";
  return "form";
}

function isJsonObject(value: unknown): value is JsonObject {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function toAPIJsonObject(value: unknown): API.JsonObject {
  if (!isJsonObject(value)) return {};
  const result: API.JsonObject = {};
  for (const [key, item] of Object.entries(value)) {
    if (
      item === undefined ||
      typeof item === "function" ||
      typeof item === "symbol"
    ) {
      continue;
    }
    result[key] = toAPIJsonValue(item);
  }
  return result;
}

function toAPIJsonValue(value: unknown): API.JsonValue {
  switch (typeof value) {
    case "string":
    case "boolean":
      return value;
    case "number":
      return Number.isFinite(value) ? value : null;
    case "object":
      if (value === null) return null;
      if (Array.isArray(value)) return value.map(toAPIJsonValue);
      return toAPIJsonObject(value);
    default:
      return null;
  }
}

function fieldString(value: unknown) {
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return "";
}

function optionalInteger(min = 0, max?: number) {
  let schema = z.number().int().min(min);
  if (max !== undefined) schema = schema.max(max);
  return z.preprocess(parseOptionalInteger, schema.optional());
}

const optionalPort = optionalInteger(1, 65_535);
const optionalNonNegativeInteger = optionalInteger();
const optionalRealityXver = optionalInteger(0, 2);
const optionalHysteriaVersion = z.preprocess(
  parseOptionalInteger,
  z.number().int().min(2).max(2).optional()
);
const optionalProxyProtocol = optionalInteger(0, 2);

const formSchema = z.object({
  name: z.string().trim().min(1).max(100),
  type: z.enum(["inbound", "outbound", "dns", "routing", "geodata"]),
  description: z.string().max(255).optional(),
  enabled: z.boolean(),
  config: z.record(z.string(), z.unknown()).optional(),
  config_template: z.string().optional(),
  variables_schema_json: z.string().optional(),
  default_variables_json: z.string().optional(),
  subscription_meta_json: z.string().optional(),
  tag: z.string().optional(),
  listen: z.string().optional(),
  port: optionalPort,
  protocol: z.string().optional(),
  settings_json: z.string().optional(),
  stream_extra_json: z.string().optional(),
  network: z.string().optional(),
  security: z.string().optional(),
  host: z.string().optional(),
  path: z.string().optional(),
  service_name: z.string().optional(),
  grpc_authority: z.string().optional(),
  grpc_multi_mode: z.boolean().optional(),
  grpc_user_agent: z.string().optional(),
  grpc_idle_timeout: optionalNonNegativeInteger,
  grpc_health_check_timeout: optionalNonNegativeInteger,
  grpc_permit_without_stream: z.boolean().optional(),
  grpc_initial_windows_size: optionalNonNegativeInteger,
  xhttp_mode: z.string().optional(),
  xhttp_extra_json: z.string().optional(),
  sni: z.string().optional(),
  allow_insecure: z.boolean().optional(),
  pinned_peer_cert_sha256: z.string().optional(),
  verify_peer_cert_by_name: z.string().optional(),
  fingerprint: z.string().optional(),
  kcp_mtu: optionalNonNegativeInteger,
  kcp_tti: optionalNonNegativeInteger,
  kcp_uplink_capacity: optionalNonNegativeInteger,
  kcp_downlink_capacity: optionalNonNegativeInteger,
  kcp_congestion: z.boolean().optional(),
  kcp_header_type: z.string().optional(),
  raw_settings_json: z.string().optional(),
  kcp_settings_json: z.string().optional(),
  hysteria_settings_json: z.string().optional(),
  hysteria_auth: z.string().optional(),
  udp_idle_timeout: optionalNonNegativeInteger,
  hysteria_masquerade_json: z.string().optional(),
  sockopt_json: z.string().optional(),
  finalmask_json: z.string().optional(),
  reality_show: z.boolean().optional(),
  reality_target: z.string().optional().refine(isRealityTarget, {
    message: "REALITY Target 必须使用 host:port 格式，例如 example.com:443。",
  }),
  reality_xver: optionalRealityXver,
  reality_server_names: z.string().optional(),
  reality_private_key: z.string().optional(),
  reality_public_key: z.string().optional(),
  reality_short_id: z
    .string()
    .optional()
    .refine((value) => !value || isRealityShortId(value), {
      message: "REALITY Short ID 必须为偶数位十六进制，最长 16 位。",
    }),
  reality_short_ids: z.string().optional().refine(isRealityShortIdList, {
    message: "REALITY Short IDs 必须为偶数位十六进制，最长 16 位。",
  }),
  reality_spider_x: z.string().optional(),
  reality_mldsa65_seed: z.string().optional(),
  reality_mldsa65_verify: z.string().optional(),
  reality_min_client_ver: z.string().optional().refine(isXrayVersion, {
    message: "最低客户端版本必须使用 x.y.z 格式。",
  }),
  reality_max_client_ver: z.string().optional().refine(isXrayVersion, {
    message: "最高客户端版本必须使用 x.y.z 格式。",
  }),
  reality_max_time_diff: optionalNonNegativeInteger,
  sniffing: z.boolean().optional(),
  dest_override: z.string().optional(),
  sniffing_metadata_only: z.boolean().optional(),
  sniffing_route_only: z.boolean().optional(),
  sniffing_domains_excluded: z.string().optional(),
  vless_clients_json: z.string().optional(),
  vless_decryption: z.string().optional(),
  vmess_clients_json: z.string().optional(),
  vmess_default_json: z.string().optional(),
  trojan_clients_json: z.string().optional(),
  fallbacks_json: z.string().optional(),
  ss_network: z.string().optional(),
  ss_method: z.string().optional(),
  ss_password: z.string().optional(),
  ss_clients_json: z.string().optional(),
  socks_auth: z.string().optional(),
  socks_udp: z.boolean().optional(),
  socks_ip: z.string().optional(),
  accounts_json: z.string().optional(),
  allow_transparent: z.boolean().optional(),
  dokodemo_address: z.string().optional(),
  dokodemo_port: optionalInteger(0, 65_535),
  dokodemo_port_map_json: z.string().optional(),
  dokodemo_network: z.string().optional(),
  follow_redirect: z.boolean().optional(),
  wg_secret_key: z.string().optional(),
  wg_address: z.string().optional(),
  wg_peers_json: z.string().optional(),
  wg_mtu: optionalNonNegativeInteger,
  wg_no_kernel_tun: z.boolean().optional(),
  wg_reserved_json: z.string().optional(),
  wg_workers: optionalNonNegativeInteger,
  wg_domain_strategy: z.string().optional(),
  hysteria_version: optionalHysteriaVersion,
  hysteria_users_json: z.string().optional(),
  tun_name: z.string().optional(),
  tun_mtu: optionalNonNegativeInteger,
  out_address: z.string().optional(),
  out_port: optionalPort,
  out_id: z.string().optional(),
  out_encryption: z.string().optional(),
  out_flow: z.string().optional(),
  out_security: z.string().optional(),
  out_experiments: z.string().optional(),
  out_password: z.string().optional(),
  out_user: z.string().optional(),
  out_pass: z.string().optional(),
  ss_uot: z.boolean().optional(),
  ss_uot_version: optionalNonNegativeInteger,
  headers_json: z.string().optional(),
  email: z.string().optional(),
  user_level: optionalNonNegativeInteger,
  domain_strategy: z.string().optional(),
  redirect: z.string().optional(),
  fragment_json: z.string().optional(),
  noises_json: z.string().optional(),
  proxy_protocol: optionalProxyProtocol,
  response_type: z.string().optional(),
  dns_out_network: z.string().optional(),
  dns_out_address: z.string().optional(),
  dns_out_port: optionalPort,
  dns_out_rules_json: z.string().optional(),
  loopback_inbound_tag: z.string().optional(),
  servers: z.string().optional(),
  dns_servers_json: z.string().optional(),
  serve_stale: z.boolean().optional(),
  serve_expired_ttl: optionalNonNegativeInteger,
  enable_parallel_query: z.boolean().optional(),
  use_system_hosts: z.boolean().optional(),
  hosts_json: z.string().optional(),
  query_strategy: z.string().optional(),
  disable_cache: z.boolean().optional(),
  disable_fallback: z.boolean().optional(),
  disable_fallback_if_match: z.boolean().optional(),
  client_ip: z.string().optional(),
  routing_domain_strategy: z.string().optional(),
  routing_rules_json: z.string().optional(),
  routing_balancers_json: z.string().optional(),
  geodata_cron: z.string().optional().refine(isFiveFieldCron, {
    message: "Cron 表达式必须包含 5 个字段，例如 0 4 * * *。",
  }),
  geodata_outbound: z.string().optional(),
  geodata_assets_json: z.string().optional(),
  advanced_json: z.string().optional(),
});

const advancedFormSchema = formSchema.pick({
  name: true,
  type: true,
  description: true,
  enabled: true,
  config_template: true,
  variables_schema_json: true,
  default_variables_json: true,
  subscription_meta_json: true,
  advanced_json: true,
});

type FormValues = z.infer<typeof formSchema>;

type TemplatePayload = {
  name: string;
  type: XrayTemplateType;
  description?: string;
  enabled?: boolean;
  config: API.JsonObject;
  config_template?: string;
  variables_schema?: API.JsonObject;
  default_variables?: API.JsonObject;
  subscription_meta?: API.JsonObject;
};

type JsonFieldRule = {
  name: FormRootFieldName;
  label: string;
  shape: "array" | "object";
};

const REALITY_NETWORKS = ["raw", "xhttp", "grpc"];
const XHTTP_MODES = ["auto", "packet-up", "stream-up", "stream-one"];
const SERVER_OUTBOUND_PROTOCOLS = [
  "vless",
  "vmess",
  "trojan",
  "shadowsocks",
  "socks",
  "http",
  "hysteria",
];

const ROUTING_RULE_COLUMNS: JsonArrayColumn[] = [
  { key: "type", label: "类型", type: "select", options: ["field"] },
  {
    key: "inboundTag",
    label: "入站 Tag",
    type: "csv",
    placeholder: "{{ .Ref.inbound.main.tag }}",
    multiline: true,
  },
  { key: "outboundTag", label: "出站 Tag", placeholder: "direct" },
  { key: "balancerTag", label: "负载均衡 Tag", placeholder: "auto" },
  {
    key: "domain",
    label: "域名规则",
    type: "csv",
    placeholder: "geosite:cn\ndomain:example.com",
    multiline: true,
    splitMode: "line",
    decodeJsonStringEscapes: true,
  },
  {
    key: "ip",
    label: "IP 规则",
    type: "csv",
    placeholder: "geoip:private\n192.168.0.0/16",
    multiline: true,
  },
  {
    key: "protocol",
    label: "协议",
    type: "csv",
    placeholder: "bittorrent",
    multiline: true,
  },
  { key: "port", label: "端口", placeholder: "25,465,587" },
];

const ROUTING_BALANCER_COLUMNS: JsonArrayColumn[] = [
  { key: "tag", label: "Tag", placeholder: "auto" },
  {
    key: "selector",
    label: "Selector",
    type: "csv",
    placeholder: "proxy\nfallback",
    multiline: true,
  },
  { key: "strategy", label: "策略", placeholder: "leastPing" },
];

const GEODATA_ASSET_COLUMNS: JsonArrayColumn[] = [
  {
    key: "url",
    label: "下载地址",
    placeholder: "https://example.com/geoip.dat",
    span: "full",
  },
  { key: "file", label: "文件名", placeholder: "geoip.dat" },
];

const DNS_SERVER_OBJECT_COLUMNS: JsonArrayColumn[] = [
  {
    key: "address",
    label: "地址",
    placeholder: "https://dns.google/dns-query",
    span: "full",
  },
  {
    key: "domains",
    label: "匹配域名",
    type: "csv",
    placeholder: "geosite:cn\ndomain:example.com",
    multiline: true,
    splitMode: "line",
    decodeJsonStringEscapes: true,
  },
  {
    key: "expectedIPs",
    label: "期望 IP",
    type: "csv",
    placeholder: "geoip:cn\n1.1.1.1",
    multiline: true,
  },
  {
    key: "queryStrategy",
    label: "查询策略",
    type: "select",
    options: [...QUERY_STRATEGIES],
  },
  { key: "skipFallback", label: "跳过 fallback", type: "boolean" },
];

const VLESS_CLIENT_COLUMNS: JsonArrayColumn[] = [
  { key: "id", label: "UUID", placeholder: "{{ .Vars.uuid }}" },
  { key: "email", label: "Email", placeholder: "{{ .Vars.email }}" },
  { key: "flow", label: "Flow", placeholder: "xtls-rprx-vision" },
  { key: "level", label: "Level", type: "integer" },
];

const VMESS_CLIENT_COLUMNS: JsonArrayColumn[] = [
  { key: "id", label: "UUID", placeholder: "{{ .Vars.uuid }}" },
  { key: "email", label: "Email", placeholder: "{{ .Vars.email }}" },
  { key: "alterId", label: "Alter ID", type: "integer" },
  { key: "level", label: "Level", type: "integer" },
];

const TROJAN_CLIENT_COLUMNS: JsonArrayColumn[] = [
  { key: "password", label: "密码", placeholder: "{{ .Vars.password }}" },
  { key: "email", label: "Email", placeholder: "{{ .Vars.email }}" },
  { key: "flow", label: "Flow" },
  { key: "level", label: "Level", type: "integer" },
];

const FALLBACK_COLUMNS: JsonArrayColumn[] = [
  { key: "dest", label: "目标", placeholder: "80 / 127.0.0.1:8443" },
  { key: "name", label: "SNI 名称", placeholder: "example.com" },
  { key: "alpn", label: "ALPN", placeholder: "h2,http/1.1" },
  { key: "path", label: "Path", placeholder: "/fallback" },
];

const SHADOWSOCKS_CLIENT_COLUMNS: JsonArrayColumn[] = [
  { key: "password", label: "密码" },
  { key: "method", label: "Method", placeholder: "aes-256-gcm" },
  { key: "email", label: "Email" },
  { key: "level", label: "Level", type: "integer" },
];

const ACCOUNT_COLUMNS: JsonArrayColumn[] = [
  { key: "user", label: "用户名" },
  { key: "pass", label: "密码" },
];

const WG_PEER_COLUMNS: JsonArrayColumn[] = [
  { key: "publicKey", label: "Public Key" },
  { key: "endpoint", label: "Endpoint", placeholder: "example.com:51820" },
  {
    key: "allowedIPs",
    label: "Allowed IPs",
    type: "csv",
    placeholder: "0.0.0.0/0\n::/0",
    multiline: true,
  },
  { key: "keepAlive", label: "Keep Alive", type: "integer" },
];

const HYSTERIA_USER_COLUMNS: JsonArrayColumn[] = [
  { key: "auth", label: "Auth", placeholder: "{{ .Vars.password }}" },
  { key: "email", label: "Email", placeholder: "{{ .Vars.email }}" },
  { key: "level", label: "Level", type: "integer" },
];

const DNS_OUT_RULE_COLUMNS: JsonArrayColumn[] = [
  { key: "action", label: "动作", placeholder: "reject / direct" },
  {
    key: "domain",
    label: "域名",
    type: "csv",
    placeholder: "domain:example.com\ngeosite:cn",
    multiline: true,
    splitMode: "line",
    decodeJsonStringEscapes: true,
  },
  {
    key: "ip",
    label: "IP",
    type: "csv",
    placeholder: "geoip:cn\n1.1.1.1",
    multiline: true,
  },
  { key: "qtype", label: "QType", type: "integer" },
];

const FREEDOM_NOISE_COLUMNS: JsonArrayColumn[] = [
  { key: "type", label: "类型", placeholder: "base64" },
  { key: "packet", label: "Packet" },
  { key: "delay", label: "延迟", placeholder: "10-16" },
];

function defaultValues(type: XrayTemplateType = "inbound"): FormValues {
  return {
    name: "",
    type,
    description: "",
    enabled: true,
    config: {},
    config_template: "",
    variables_schema_json: "{}",
    default_variables_json: "{}",
    subscription_meta_json: "{}",
    protocol: type === "outbound" ? "freedom" : "vless",
    tag: "",
    listen: "",
    port: undefined,
    settings_json: "{}",
    stream_extra_json: "{}",
    network: "raw",
    security: "none",
    fingerprint: "chrome",
    pinned_peer_cert_sha256: "",
    verify_peer_cert_by_name: "",
    grpc_multi_mode: false,
    grpc_user_agent: "",
    grpc_permit_without_stream: false,
    xhttp_mode: "auto",
    xhttp_extra_json: "{}",
    kcp_congestion: false,
    kcp_header_type: "",
    reality_show: false,
    reality_target: "",
    reality_server_names: "",
    reality_private_key: "",
    reality_public_key: "",
    reality_short_id: "",
    reality_short_ids: "",
    reality_spider_x: "",
    reality_mldsa65_seed: "",
    reality_mldsa65_verify: "",
    raw_settings_json: "{}",
    kcp_settings_json: "{}",
    hysteria_settings_json: "{}",
    hysteria_auth: "",
    udp_idle_timeout: 60,
    hysteria_masquerade_json: "{}",
    sockopt_json: "{}",
    finalmask_json: "{}",
    vless_clients_json: "[]",
    vless_decryption: "none",
    vmess_clients_json: "[]",
    vmess_default_json: "{}",
    trojan_clients_json: "[]",
    fallbacks_json: "[]",
    ss_network: "tcp",
    ss_method: "",
    ss_password: "",
    ss_clients_json: "[]",
    socks_auth: "noauth",
    socks_udp: false,
    accounts_json: "[]",
    allow_transparent: false,
    dokodemo_address: "localhost",
    dokodemo_port: undefined,
    dokodemo_port_map_json: "{}",
    dokodemo_network: "tcp",
    follow_redirect: false,
    wg_secret_key: "",
    wg_address: "10.0.0.1\nfd59:7153:2388:b5fd::1",
    wg_peers_json: "[]",
    wg_mtu: 1420,
    wg_no_kernel_tun: false,
    wg_reserved_json: "[]",
    wg_domain_strategy: "ForceIP",
    hysteria_version: 2,
    hysteria_users_json: "[]",
    tun_name: "xray0",
    tun_mtu: 1500,
    out_encryption: "none",
    out_security: "auto",
    ss_uot: false,
    ss_uot_version: 2,
    headers_json: "{}",
    fragment_json: "{}",
    noises_json: "[]",
    user_level: 0,
    domain_strategy: "AsIs",
    response_type: "none",
    dns_out_network: "udp",
    dns_out_port: 53,
    dns_out_rules_json: "[]",
    sniffing: true,
    dest_override: "http, tls, quic",
    sniffing_metadata_only: false,
    sniffing_route_only: false,
    sniffing_domains_excluded: "",
    query_strategy: "UseIP",
    hosts_json: "{}",
    servers: "1.1.1.1\n8.8.8.8",
    dns_servers_json: "[]",
    serve_stale: false,
    serve_expired_ttl: undefined,
    enable_parallel_query: false,
    use_system_hosts: false,
    routing_domain_strategy: "AsIs",
    routing_rules_json: "[]",
    routing_balancers_json: "[]",
    geodata_cron: "0 4 * * *",
    geodata_outbound: "",
    geodata_assets_json:
      '[\n  {\n    "url": "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",\n    "file": "geoip.dat"\n  },\n  {\n    "url": "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat",\n    "file": "geosite.dat"\n  }\n]',
    advanced_json: "{}",
  };
}

function JsonTextarea({
  value,
  onChange,
  placeholder,
}: {
  value?: string;
  onChange: (value: string) => void;
  placeholder?: string;
}) {
  return (
    <Textarea
      className="min-h-32 font-mono text-xs"
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      value={value || ""}
    />
  );
}

function SummaryPill({
  label,
  value,
}: {
  label: string;
  value?: string | number | boolean;
}) {
  return (
    <div className="xray-summary-item">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="truncate font-medium text-sm">
        {value === undefined || value === "" ? "—" : String(value)}
      </div>
    </div>
  );
}

function FormSectionHeading({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="xray-form-section-heading">
      <div className="font-semibold text-sm">{title}</div>
      <p className="text-muted-foreground text-xs leading-5">{description}</p>
    </div>
  );
}

function TabErrorCount({ count }: { count: number }) {
  if (!count) return null;
  return (
    <span
      className="inline-flex size-4 shrink-0 items-center justify-center rounded-full bg-destructive text-[10px] text-destructive-foreground"
      title={`${count} errors`}
    >
      {count > 9 ? "9+" : count}
    </span>
  );
}

function jsonArrayCount(value?: string) {
  const parsed = safeJsonParse<unknown>(value || "", []);
  return Array.isArray(parsed) ? parsed.length : 0;
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

function stableJsonKey(value: unknown) {
  return JSON.stringify(sortJsonValue(value));
}

function findDefaultVariableConflicts(
  type: XrayTemplateType,
  config: JsonObject,
  defaultVariables: JsonObject
) {
  const configVariables = extractTemplateConfigVariables({ type, config });
  return Object.keys(defaultVariables).filter(
    (key) =>
      Object.hasOwn(configVariables, key) &&
      stableJsonKey(configVariables[key]) !==
        stableJsonKey(defaultVariables[key])
  );
}

function objectJsonField(
  name: FormRootFieldName,
  label: string
): JsonFieldRule {
  return { name, label, shape: "object" };
}

function arrayJsonField(name: FormRootFieldName, label: string): JsonFieldRule {
  return { name, label, shape: "array" };
}

function pushUniqueJsonField(
  map: Map<FormRootFieldName, JsonFieldRule>,
  rule: JsonFieldRule
) {
  map.set(rule.name, rule);
}

function getActiveJsonFieldRules(values: FormValues, useAdvancedJson: boolean) {
  const rules = new Map<FormRootFieldName, JsonFieldRule>();
  const add = (rule: JsonFieldRule) => pushUniqueJsonField(rules, rule);

  add(objectJsonField("variables_schema_json", "Variables Schema"));
  add(objectJsonField("default_variables_json", "Default Variables"));
  add(objectJsonField("subscription_meta_json", "Subscription Meta"));
  if (useAdvancedJson) {
    add(objectJsonField("advanced_json", "Advanced JSON"));
    return [...rules.values()];
  }

  if (values.type === "routing") {
    add(arrayJsonField("routing_rules_json", "Routing Rules"));
    add(arrayJsonField("routing_balancers_json", "Routing Balancers"));
    return [...rules.values()];
  }

  if (values.type === "dns") {
    add(arrayJsonField("dns_servers_json", "DNS Object Servers"));
    add(objectJsonField("hosts_json", "Hosts JSON"));
    return [...rules.values()];
  }

  if (values.type === "geodata") {
    add(arrayJsonField("geodata_assets_json", "Geodata Assets"));
    return [...rules.values()];
  }

  add(objectJsonField("settings_json", "Settings JSON"));

  if (values.protocol === "vless") {
    add(arrayJsonField("vless_clients_json", "VLESS Users"));
    add(arrayJsonField("fallbacks_json", "Fallbacks"));
  }
  if (values.protocol === "vmess") {
    add(arrayJsonField("vmess_clients_json", "VMess Users"));
    add(objectJsonField("vmess_default_json", "VMess Default"));
  }
  if (values.protocol === "trojan") {
    add(arrayJsonField("trojan_clients_json", "Trojan Users"));
    add(arrayJsonField("fallbacks_json", "Fallbacks"));
  }
  if (values.protocol === "shadowsocks") {
    add(arrayJsonField("ss_clients_json", "Shadowsocks Users"));
  }
  if (values.protocol === "socks" || values.protocol === "http") {
    add(arrayJsonField("accounts_json", "Accounts"));
  }
  if (values.protocol === "tunnel") {
    add(objectJsonField("dokodemo_port_map_json", "Port Map"));
  }
  if (values.protocol === "wireguard") {
    add(arrayJsonField("wg_peers_json", "WireGuard Peers"));
  }
  if (values.protocol === "hysteria") {
    add(arrayJsonField("hysteria_users_json", "Hysteria Users"));
  }

  if (values.type === "outbound") {
    if (values.protocol === "freedom") {
      add(objectJsonField("fragment_json", "Fragment"));
      add(arrayJsonField("noises_json", "Noises"));
    }
    if (values.protocol === "http") {
      add(objectJsonField("headers_json", "Headers"));
    }
    if (values.protocol === "dns") {
      add(arrayJsonField("dns_out_rules_json", "DNS Rules"));
    }
    if (values.protocol === "wireguard") {
      add(arrayJsonField("wg_reserved_json", "WireGuard Reserved"));
    }
  }

  const streamUnsupported =
    (values.type === "inbound" &&
      ["tun", "wireguard"].includes(values.protocol || "")) ||
    (values.type === "outbound" &&
      ["dns", "loopback", "wireguard"].includes(values.protocol || ""));
  if (!streamUnsupported) {
    if (values.network === "xhttp") {
      add(objectJsonField("xhttp_extra_json", "XHTTP Extra"));
    }
    if (values.network === "raw") {
      add(objectJsonField("raw_settings_json", "Raw Settings"));
    }
    if (values.network === "mkcp") {
      add(objectJsonField("kcp_settings_json", "KCP Settings"));
    }
    if (values.network === "hysteria") {
      add(objectJsonField("hysteria_settings_json", "Hysteria Settings"));
    }
    add(objectJsonField("sockopt_json", "Sockopt"));
    add(objectJsonField("finalmask_json", "Finalmask"));
  }

  return [...rules.values()];
}

function parseJsonField(value: unknown) {
  const text = typeof value === "string" ? value.trim() : "";
  if (!text) return;
  return JSON.parse(text);
}

function isPlainJsonObject(value: unknown): value is JsonObject {
  return isJsonObject(value);
}

const NON_CONFIG_DIRTY_FIELDS = new Set<FormRootFieldName>([
  "name",
  "description",
  "enabled",
  "config",
  "config_template",
  "variables_schema_json",
  "default_variables_json",
  "subscription_meta_json",
  "advanced_json",
]);

function hasStructuredConfigChanges(dirtyFields: Record<string, unknown>) {
  return Object.keys(dirtyFields || {}).some(
    (key) => !NON_CONFIG_DIRTY_FIELDS.has(key as FormRootFieldName)
  );
}

function fieldLabelKey(label: string) {
  return label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

function useXrayFieldLabel(label: string) {
  const { t } = useTranslation("xray-templates");
  const key = fieldLabelKey(label);
  if (!key) return label;
  return t(`fieldLabels.${key}`, label);
}

function CompositeFieldLabel({ children }: { children: ReactNode }) {
  return <div className="font-medium text-sm">{children}</div>;
}

function SwitchField({
  control,
  name,
  label,
  description,
}: {
  control: XrayFormControl;
  name: FormFieldName;
  label: string;
  description?: string;
}) {
  const translatedLabel = useXrayFieldLabel(label);

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className="xray-switch-field">
          <div className="min-w-0">
            <FormLabel>
              <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
            </FormLabel>
            <XrayFieldDescription description={description} fieldKey={name} />
          </div>
          <FormControl>
            <Switch
              aria-label={translatedLabel}
              checked={!!field.value}
              onCheckedChange={field.onChange}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function InputField({
  control,
  name,
  label,
  placeholder,
  type,
  description,
  required,
}: {
  control: XrayFormControl;
  name: FormFieldName;
  label: string;
  placeholder?: string;
  type?: string;
  description?: string;
  required?: boolean;
}) {
  const translatedLabel = useXrayFieldLabel(label);
  const isNumber = type === "number";
  const isPort = ["port", "dokodemo_port", "out_port", "dns_out_port"].includes(
    name
  );
  const allowsZeroPort = name === "dokodemo_port";
  const max = isPort
    ? 65_535
    : name === "hysteria_version"
      ? 2
      : name === "reality_xver" || name === "proxy_protocol"
        ? 2
        : undefined;
  const suffix = [
    "grpc_idle_timeout",
    "grpc_health_check_timeout",
    "udp_idle_timeout",
    "serve_expired_ttl",
  ].includes(name)
    ? "s"
    : name === "reality_max_time_diff"
      ? "ms"
      : name === "grpc_initial_windows_size"
        ? "B"
        : undefined;

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>
            <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
            {required ? (
              <span aria-hidden="true" className="ml-1 text-destructive">
                *
              </span>
            ) : null}
          </FormLabel>
          <XrayFieldDescription description={description} fieldKey={name} />
          <FormControl>
            <EnhancedInput
              inputMode={isNumber ? "numeric" : undefined}
              max={max}
              min={
                isNumber
                  ? name === "hysteria_version"
                    ? 2
                    : isPort
                      ? allowsZeroPort
                        ? 0
                        : 1
                      : 0
                  : undefined
              }
              onValueChange={field.onChange}
              placeholder={placeholder}
              preserveZero={isNumber && !isPort}
              suffix={suffix}
              step={isNumber ? 1 : undefined}
              type={type}
              value={field.value ?? ""}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function SelectField({
  control,
  name,
  label,
  options,
  description,
}: {
  control: XrayFormControl;
  name: FormFieldName;
  label: string;
  options: string[];
  description?: string;
}) {
  const translatedLabel = useXrayFieldLabel(label);

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>
            <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
          </FormLabel>
          <XrayFieldDescription description={description} fieldKey={name} />
          <Select
            onValueChange={field.onChange}
            value={fieldString(field.value)}
          >
            <FormControl>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              {options.map((item) => (
                <SelectItem key={item} value={item}>
                  {item}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function JsonField({
  control,
  name,
  label,
  description,
  placeholder,
}: {
  control: XrayFormControl;
  name: FormFieldName;
  label: string;
  description?: string;
  placeholder?: string;
}) {
  const translatedLabel = useXrayFieldLabel(label);

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>
            <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
          </FormLabel>
          <XrayFieldDescription description={description} fieldKey={name} />
          <FormControl>
            <JsonTextarea
              onChange={field.onChange}
              placeholder={placeholder}
              value={fieldString(field.value)}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function JsonObjectField({
  control,
  name,
  label,
  description,
  addLabel,
  className,
}: {
  control: XrayFormControl;
  name: FormFieldName;
  label: string;
  description?: string;
  addLabel?: string;
  className?: string;
}) {
  const translatedLabel = useXrayFieldLabel(label);

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className={className}>
          <CompositeFieldLabel>
            <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
          </CompositeFieldLabel>
          <XrayFieldDescription description={description} fieldKey={name} />
          <JsonObjectEditor
            addLabel={addLabel}
            onChange={field.onChange}
            value={fieldString(field.value)}
          />
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function JsonArrayObjectField({
  addLabel,
  className,
  columns,
  control,
  defaultItem,
  description,
  emptyText,
  label,
  name,
}: {
  addLabel?: string;
  className?: string;
  columns: JsonArrayColumn[];
  control: XrayFormControl;
  defaultItem?: JsonObject;
  description?: string;
  emptyText?: string;
  label: string;
  name: FormFieldName;
}) {
  const translatedLabel = useXrayFieldLabel(label);

  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem className={className}>
          <CompositeFieldLabel>
            <XrayFieldLabel fieldKey={name}>{translatedLabel}</XrayFieldLabel>
          </CompositeFieldLabel>
          <XrayFieldDescription description={description} fieldKey={name} />
          <JsonArrayObjectEditor
            addLabel={addLabel}
            columns={columns}
            defaultItem={defaultItem}
            emptyText={emptyText}
            onChange={field.onChange}
            value={fieldString(field.value)}
          />
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

function VariableSchemaField({
  control,
  name,
}: {
  control: XrayFormControl;
  name: FormFieldName;
}) {
  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => {
        const schema = parseJsonObjectText(fieldString(field.value));
        const schemaMetaKeys = new Set([
          "type",
          "required",
          "properties",
          "additionalProperties",
          "description",
          "title",
        ]);
        const properties =
          schema.properties && typeof schema.properties === "object"
            ? (schema.properties as JsonObject)
            : Object.fromEntries(
                Object.entries(schema).filter(
                  ([key, value]) =>
                    !schemaMetaKeys.has(key) &&
                    value &&
                    typeof value === "object" &&
                    !Array.isArray(value)
                )
              );
        const required = Array.isArray(schema.required) ? schema.required : [];
        const rows = Object.entries(properties);

        const commit = (
          nextProperties: JsonObject,
          nextRequired = required
        ) => {
          const baseSchema = { ...schema };
          if (!schema.properties) {
            for (const key of Object.keys(properties)) {
              delete baseSchema[key];
            }
          }
          field.onChange(
            formatJson({
              ...baseSchema,
              type: schema.type || "object",
              properties: nextProperties,
              required: nextRequired,
            })
          );
        };

        return (
          <FormItem>
            <CompositeFieldLabel>变量结构</CompositeFieldLabel>
            <FormDescription>
              定义绑定服务器时要填写的变量，绑定页会自动生成对应表单。
            </FormDescription>
            <div className="space-y-2 rounded-md border bg-muted/20 p-3">
              {rows.length ? (
                rows.map(([key, rawItem]) => {
                  const item = isJsonObject(rawItem) ? rawItem : {};
                  return (
                    <div
                      className="space-y-2 rounded-md border bg-background p-2"
                      key={key}
                    >
                      <div className="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,1fr)_140px]">
                        <EnhancedInput
                          onValueChange={(nextKey) => {
                            const normalized = nextKey.trim();
                            if (!normalized || normalized === key) return;
                            const next = { ...properties };
                            delete next[key];
                            next[normalized] = item;
                            commit(
                              next,
                              required.map((requiredKey: string) =>
                                requiredKey === key ? normalized : requiredKey
                              )
                            );
                          }}
                          placeholder="变量名"
                          value={key}
                        />
                        <Select
                          onValueChange={(typeValue) =>
                            commit({
                              ...properties,
                              [key]: { ...item, type: typeValue },
                            })
                          }
                          value={fieldString(item.type) || "string"}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="string">文本</SelectItem>
                            <SelectItem value="number">数字</SelectItem>
                            <SelectItem value="boolean">开关</SelectItem>
                            <SelectItem value="array">数组</SelectItem>
                            <SelectItem value="object">对象</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="grid grid-cols-1 gap-2 lg:grid-cols-3">
                        <EnhancedInput
                          onValueChange={(titleValue) =>
                            commit({
                              ...properties,
                              [key]: { ...item, title: titleValue },
                            })
                          }
                          placeholder="显示名"
                          value={
                            fieldString(item.title) || fieldString(item.label)
                          }
                        />
                        <EnhancedInput
                          onValueChange={(descriptionValue) =>
                            commit({
                              ...properties,
                              [key]: {
                                ...item,
                                description: descriptionValue,
                              },
                            })
                          }
                          placeholder="说明"
                          value={
                            fieldString(item.description) ||
                            fieldString(item.desc)
                          }
                        />
                        <EnhancedInput
                          onValueChange={(defaultValue) =>
                            commit({
                              ...properties,
                              [key]: { ...item, default: defaultValue },
                            })
                          }
                          placeholder="默认值"
                          value={fieldString(item.default)}
                        />
                      </div>
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
                          <Switch
                            checked={required.includes(key)}
                            onCheckedChange={(checked) =>
                              commit(
                                properties,
                                checked
                                  ? [...required, key]
                                  : required.filter(
                                      (requiredKey: string) =>
                                        requiredKey !== key
                                    )
                              )
                            }
                          />
                          <span className="text-muted-foreground text-xs">
                            必填
                          </span>
                        </div>
                        <Button
                          className="h-9 px-3"
                          onClick={() => {
                            const next = { ...properties };
                            delete next[key];
                            commit(
                              next,
                              required.filter(
                                (requiredKey: string) => requiredKey !== key
                              )
                            );
                          }}
                          type="button"
                          variant="outline"
                        >
                          删除
                        </Button>
                      </div>
                    </div>
                  );
                })
              ) : (
                <Empty className="border p-4">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <Braces />
                    </EmptyMedia>
                    <EmptyDescription>还没有变量定义。</EmptyDescription>
                  </EmptyHeader>
                </Empty>
              )}
              <Button
                onClick={() => {
                  const nextKey = `var_${rows.length + 1}`;
                  commit({
                    ...properties,
                    [nextKey]: { type: "string", title: nextKey },
                  });
                }}
                size="sm"
                type="button"
                variant="outline"
              >
                添加变量
              </Button>
            </div>
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}

function ProtocolSettingsFields({
  control,
  type,
  protocol,
}: {
  control: XrayFormControl;
  type: XrayTemplateType;
  protocol?: string;
}) {
  const { t } = useTranslation("xray-templates");

  if (type === "inbound") {
    if (protocol === "vless") {
      return (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <InputField
              control={control}
              label="Decryption"
              name="vless_decryption"
              placeholder="none"
            />
            <InputField
              control={control}
              label="User Level"
              name="user_level"
              type="number"
            />
          </div>
          <JsonArrayObjectField
            addLabel="添加用户"
            columns={VLESS_CLIENT_COLUMNS}
            control={control}
            defaultItem={{ id: "{{ .Vars.uuid }}", email: "{{ .Vars.email }}" }}
            description="每一行会生成一个 VLESS 用户，支持直接使用 Vars 模板变量。"
            label="VLESS Users"
            name="vless_clients_json"
          />
          <JsonArrayObjectField
            addLabel="添加回落"
            columns={FALLBACK_COLUMNS}
            control={control}
            description="按需配置 REALITY/TLS 回落目标，留空则不生成 fallbacks。"
            label="Fallbacks"
            name="fallbacks_json"
          />
        </div>
      );
    }
    if (protocol === "vmess") {
      return (
        <div className="space-y-4">
          <JsonArrayObjectField
            addLabel="添加用户"
            columns={VMESS_CLIENT_COLUMNS}
            control={control}
            defaultItem={{
              alterId: 0,
              email: "{{ .Vars.email }}",
              id: "{{ .Vars.uuid }}",
            }}
            description="每一行会生成一个 VMess 用户。"
            label="VMess Users"
            name="vmess_clients_json"
          />
          <JsonObjectField
            control={control}
            description="VMess default 对象，常用字段可直接新增键值。"
            label="VMess Default"
            name="vmess_default_json"
          />
        </div>
      );
    }
    if (protocol === "trojan") {
      return (
        <div className="space-y-4">
          <JsonArrayObjectField
            addLabel="添加用户"
            columns={TROJAN_CLIENT_COLUMNS}
            control={control}
            defaultItem={{
              email: "{{ .Vars.email }}",
              password: "{{ .Vars.password }}",
            }}
            description="每一行会生成一个 Trojan 用户。"
            label="Trojan Users"
            name="trojan_clients_json"
          />
          <JsonArrayObjectField
            addLabel="添加回落"
            columns={FALLBACK_COLUMNS}
            control={control}
            description="按需配置回落目标，留空则不生成 fallbacks。"
            label="Fallbacks"
            name="fallbacks_json"
          />
        </div>
      );
    }
    if (protocol === "shadowsocks") {
      return (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <SelectField
              control={control}
              label="Network"
              name="ss_network"
              options={["tcp", "udp", "tcp,udp"]}
            />
            <InputField
              control={control}
              label="Method"
              name="ss_method"
              placeholder="2022-blake3-aes-128-gcm"
            />
            <InputField control={control} label="Password" name="ss_password" />
            <InputField
              control={control}
              label="Email"
              name="email"
              placeholder="user@example.com"
            />
            <InputField
              control={control}
              label="Level"
              name="user_level"
              type="number"
            />
          </div>
          <JsonArrayObjectField
            addLabel="添加用户"
            columns={SHADOWSOCKS_CLIENT_COLUMNS}
            control={control}
            description="多用户 Shadowsocks 用户；单用户场景可只填写上方密码。"
            label="Users"
            name="ss_clients_json"
          />
        </div>
      );
    }
    if (protocol === "socks") {
      return (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <SelectField
            control={control}
            label="Auth"
            name="socks_auth"
            options={["noauth", "password"]}
          />
          <SwitchField control={control} label="UDP" name="socks_udp" />
          <InputField control={control} label="UDP IP" name="socks_ip" />
          <InputField
            control={control}
            label="User Level"
            name="user_level"
            type="number"
          />
          <div className="md:col-span-2">
            <JsonArrayObjectField
              addLabel="添加账号"
              columns={ACCOUNT_COLUMNS}
              control={control}
              description="SOCKS 用户名密码账号。"
              label="Accounts"
              name="accounts_json"
            />
          </div>
        </div>
      );
    }
    if (protocol === "http") {
      return (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <SwitchField
              control={control}
              label="Allow Transparent"
              name="allow_transparent"
            />
            <InputField
              control={control}
              label="User Level"
              name="user_level"
              type="number"
            />
          </div>
          <JsonArrayObjectField
            addLabel="添加账号"
            columns={ACCOUNT_COLUMNS}
            control={control}
            description="HTTP 用户名密码账号。"
            label="Accounts"
            name="accounts_json"
          />
        </div>
      );
    }
    if (protocol === "tunnel") {
      return (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <InputField
            control={control}
            label="Address"
            name="dokodemo_address"
          />
          <InputField
            control={control}
            description="0 表示沿用当前监听端口。"
            label="Target Port"
            name="dokodemo_port"
            type="number"
          />
          <SelectField
            control={control}
            label="Network"
            name="dokodemo_network"
            options={["tcp", "udp", "tcp,udp"]}
          />
          <SwitchField
            control={control}
            label="Follow Redirect"
            name="follow_redirect"
          />
          <InputField
            control={control}
            label="User Level"
            name="user_level"
            type="number"
          />
          <div className="md:col-span-2">
            <JsonObjectField
              control={control}
              description="Port Map 键为监听端口，值为转发目标。"
              label="Port Map"
              name="dokodemo_port_map_json"
            />
          </div>
        </div>
      );
    }
    if (protocol === "wireguard") {
      return (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <InputField
              control={control}
              label="Secret Key"
              name="wg_secret_key"
            />
            <InputField
              control={control}
              label="MTU"
              name="wg_mtu"
              type="number"
            />
          </div>
          <JsonArrayObjectField
            addLabel="添加 Peer"
            columns={WG_PEER_COLUMNS}
            control={control}
            description="WireGuard Peer 列表。"
            label="Peers"
            name="wg_peers_json"
          />
        </div>
      );
    }
    if (protocol === "hysteria") {
      return (
        <div className="space-y-4">
          <InputField
            control={control}
            description="官方要求固定为 2。"
            label="Version"
            name="hysteria_version"
            required
            type="number"
          />
          <JsonArrayObjectField
            addLabel="添加用户"
            columns={HYSTERIA_USER_COLUMNS}
            control={control}
            description="Hysteria2 用户列表。"
            label="Users"
            name="hysteria_users_json"
          />
        </div>
      );
    }
    if (protocol === "tun") {
      return (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <InputField control={control} label="Name" name="tun_name" />
          <InputField
            control={control}
            label="MTU"
            name="tun_mtu"
            type="number"
          />
          <InputField
            control={control}
            label="User Level"
            name="user_level"
            type="number"
          />
        </div>
      );
    }
  }

  if (type === "outbound" && protocol === "dns") {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <SelectField
            control={control}
            label="Network"
            name="dns_out_network"
            options={["tcp", "udp"]}
          />
          <InputField
            control={control}
            label="Address"
            name="dns_out_address"
            placeholder="1.1.1.1"
          />
          <InputField
            control={control}
            label="Port"
            name="dns_out_port"
            type="number"
          />
          <InputField
            control={control}
            label="User Level"
            name="user_level"
            type="number"
          />
        </div>
        <JsonArrayObjectField
          addLabel="添加规则"
          columns={DNS_OUT_RULE_COLUMNS}
          control={control}
          description="DNS 出站规则。"
          label="Rules"
          name="dns_out_rules_json"
        />
      </div>
    );
  }

  if (type === "outbound" && protocol === "loopback") {
    return (
      <InputField
        control={control}
        description="重新送入路由时使用的 inboundTag。"
        label="Inbound Tag"
        name="loopback_inbound_tag"
      />
    );
  }

  if (type === "outbound" && protocol === "wireguard") {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <InputField
            control={control}
            label="Secret Key"
            name="wg_secret_key"
          />
          <InputField
            control={control}
            label="MTU"
            name="wg_mtu"
            type="number"
          />
          <SwitchField
            control={control}
            label="No Kernel TUN"
            name="wg_no_kernel_tun"
          />
          <InputField
            control={control}
            label="Workers"
            name="wg_workers"
            type="number"
          />
          <InputField
            control={control}
            label="Domain Strategy"
            name="wg_domain_strategy"
            placeholder="ForceIP"
          />
        </div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <FormField
            control={control}
            name="wg_address"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("fieldLabels.address", "Address")}</FormLabel>
                <FormDescription>一行一个 WireGuard 本地地址。</FormDescription>
                <FormControl>
                  <Textarea
                    className="min-h-24"
                    onChange={field.onChange}
                    value={field.value || ""}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <JsonField
            control={control}
            description="reserved array: [1,2,3]"
            label="Reserved"
            name="wg_reserved_json"
          />
        </div>
        <JsonArrayObjectField
          addLabel="添加 Peer"
          columns={WG_PEER_COLUMNS}
          control={control}
          description="WireGuard Peer 列表。"
          label="Peers"
          name="wg_peers_json"
        />
      </div>
    );
  }

  if (type === "outbound" && protocol === "hysteria") {
    return (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <InputField
          control={control}
          description="官方要求固定为 2。"
          label="Version"
          name="hysteria_version"
          required
          type="number"
        />
        <InputField
          control={control}
          label="Address"
          name="out_address"
          required
        />
        <InputField
          control={control}
          label="Port"
          name="out_port"
          required
          type="number"
        />
      </div>
    );
  }

  if (
    type === "outbound" &&
    ["vless", "vmess", "trojan", "shadowsocks", "socks", "http"].includes(
      protocol || ""
    )
  ) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <InputField
            control={control}
            label="Address"
            name="out_address"
            required
          />
          <InputField
            control={control}
            label="Port"
            name="out_port"
            required
            type="number"
          />
          <InputField
            control={control}
            label="Level"
            name="user_level"
            type="number"
          />
          {["vless", "vmess"].includes(protocol || "") ? (
            <InputField control={control} label="ID" name="out_id" required />
          ) : null}
          {protocol === "vless" ? (
            <>
              <InputField
                control={control}
                label="Encryption"
                name="out_encryption"
                placeholder="none"
              />
              <InputField
                control={control}
                label="Flow"
                name="out_flow"
                placeholder="xtls-rprx-vision"
              />
            </>
          ) : null}
          {protocol === "vmess" ? (
            <>
              <InputField
                control={control}
                label="Security"
                name="out_security"
                placeholder="auto"
              />
              <InputField
                control={control}
                label="Experiments"
                name="out_experiments"
              />
            </>
          ) : null}
          {protocol === "trojan" ? (
            <InputField
              control={control}
              label="Password"
              name="out_password"
              required
            />
          ) : null}
          {protocol === "shadowsocks" ? (
            <>
              <InputField
                control={control}
                label="Method"
                name="ss_method"
                placeholder="2022-blake3-aes-128-gcm"
                required
              />
              <InputField
                control={control}
                label="Password"
                name="ss_password"
                required
              />
              <SwitchField control={control} label="UoT" name="ss_uot" />
              <InputField
                control={control}
                label="UoT Version"
                name="ss_uot_version"
                type="number"
              />
            </>
          ) : null}
          {["socks", "http"].includes(protocol || "") ? (
            <>
              <InputField control={control} label="User" name="out_user" />
              <InputField control={control} label="Password" name="out_pass" />
            </>
          ) : null}
          {["trojan", "shadowsocks", "socks", "http"].includes(
            protocol || ""
          ) ? (
            <InputField
              control={control}
              label="Email"
              name="email"
              placeholder="user@example.com"
            />
          ) : null}
        </div>
        {protocol === "http" ? (
          <JsonObjectField
            control={control}
            description="HTTP Header 键值。"
            label="Headers"
            name="headers_json"
          />
        ) : null}
      </div>
    );
  }

  return null;
}

export default function XrayTemplateForm({
  trigger,
  title,
  loading,
  initialValues,
  loadInitialValues,
  onSubmit,
}: {
  trigger: React.ReactNode;
  title: string;
  loading?: boolean;
  initialValues?: Partial<API.XrayTemplate>;
  loadInitialValues?: () => Promise<Partial<API.XrayTemplate> | undefined>;
  onSubmit: (values: TemplatePayload) => Promise<boolean> | boolean;
}) {
  const { t } = useTranslation("xray-templates");
  const [open, setOpen] = useState(false);
  const [opening, setOpening] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [activeInitialValues, setActiveInitialValues] = useState<
    Partial<API.XrayTemplate> | undefined
  >(initialValues);
  const [advancedTouched, setAdvancedTouched] = useState(false);
  const [activeTab, setActiveTab] = useState<XrayFormTab>("form");
  const initialType = initialValues?.type || "inbound";

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema) as Resolver<FormValues>,
    defaultValues: {
      ...defaultValues(initialType),
      name: initialValues?.name || "",
      type: initialType,
      description: initialValues?.description || "",
      enabled: initialValues?.enabled ?? true,
      config: initialValues?.config || {},
      config_template: initialValues?.config_template || "",
      variables_schema_json: formatJson(initialValues?.variables_schema || {}),
      default_variables_json: formatJson(
        initialValues?.default_variables || {}
      ),
      subscription_meta_json: formatJson(
        initialValues?.subscription_meta || {}
      ),
      ...configToFormValues(initialType, initialValues?.config),
    },
  });

  const type = useWatch({ control: form.control, name: "type" });
  const protocol = useWatch({ control: form.control, name: "protocol" });
  const security = useWatch({ control: form.control, name: "security" });
  const network = useWatch({ control: form.control, name: "network" });
  const watchedValues = useWatch({ control: form.control }) as FormValues;
  const { dirtyFields, errors } = form.formState;
  const tabErrorCounts = useMemo(() => {
    const counts: Record<XrayFormTab, number> = {
      form: 0,
      advanced: 0,
      template: 0,
      subscription: 0,
    };
    for (const field of Object.keys(errors)) {
      counts[tabForField(field)] += 1;
    }
    return counts;
  }, [errors]);
  const supportsStreamSettings = !(
    (type === "inbound" && ["tun", "wireguard"].includes(protocol || "")) ||
    (type === "outbound" &&
      ["dns", "loopback", "wireguard"].includes(protocol || ""))
  );

  const protocolOptions = useMemo(
    () => (type === "outbound" ? OUTBOUND_PROTOCOLS : INBOUND_PROTOCOLS),
    [type]
  );
  const networkOptions = useMemo(
    () =>
      protocol === "hysteria"
        ? NETWORKS.filter((item) => item === "hysteria")
        : NETWORKS,
    [protocol]
  );
  const securityOptions = useMemo(
    () =>
      network === "hysteria"
        ? (["tls"] as const)
        : REALITY_NETWORKS.includes(network || "")
          ? SECURITIES
          : SECURITIES.filter((item) => item !== "reality"),
    [network]
  );

  useEffect(() => {
    if (!open) return;
    setAdvancedTouched(false);
    setActiveTab("form");
    const values = activeInitialValues || initialValues;
    const nextType = values?.type || "inbound";
    form.reset({
      ...defaultValues(nextType),
      name: values?.name || "",
      type: nextType,
      description: values?.description || "",
      enabled: values?.enabled ?? true,
      config: values?.config || {},
      config_template: values?.config_template || "",
      variables_schema_json: formatJson(values?.variables_schema || {}),
      default_variables_json: formatJson(values?.default_variables || {}),
      subscription_meta_json: formatJson(values?.subscription_meta || {}),
      ...configToFormValues(nextType, values?.config),
    });
  }, [activeInitialValues, form, initialValues, open]);

  useEffect(() => {
    if (type === "dns" || type === "routing" || type === "geodata") return;
    const options =
      type === "outbound" ? OUTBOUND_PROTOCOLS : INBOUND_PROTOCOLS;
    if (!options.some((option) => option === protocol)) {
      form.setValue("protocol", options[0]);
    }
  }, [form, protocol, type]);

  useEffect(() => {
    if (!supportsStreamSettings) return;
    if (protocol === "hysteria" && network !== "hysteria") {
      form.setValue("network", "hysteria");
    }
    if (network === "hysteria" && security !== "tls") {
      form.setValue("security", "tls", {
        shouldDirty: true,
        shouldValidate: true,
      });
      return;
    }
    if (security === "reality" && !REALITY_NETWORKS.includes(network || "")) {
      form.setValue("security", "none");
    }
  }, [form, network, protocol, security, supportsStreamSettings]);

  function applyRoutingPreset(preset: (typeof ROUTING_PRESETS)[number]) {
    const options = {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    };
    form.setValue("routing_domain_strategy", preset.domainStrategy, options);
    form.setValue("routing_rules_json", formatJson(preset.rules), options);
    form.setValue(
      "routing_balancers_json",
      formatJson(preset.balancers),
      options
    );
    form.setValue("advanced_json", "{}", options);
    setAdvancedTouched(false);
  }

  function buildConfig(values: FormValues, useAdvancedJson = advancedTouched) {
    const source = useAdvancedJson ? values : { ...values, advanced_json: "" };
    if (source.type === "routing") {
      return buildRoutingConfig(source);
    }
    if (source.type === "geodata") return buildGeodataConfig(source);
    if (source.type === "dns") return buildDnsConfig(source);
    if (source.type === "outbound") return buildOutboundConfig(source);
    return buildInboundConfig(source);
  }

  function validateJsonFields(
    values: FormValues,
    useAdvancedJson = advancedTouched
  ) {
    const rules = getActiveJsonFieldRules(values, useAdvancedJson);
    form.clearErrors(rules.map((rule) => rule.name));
    for (const rule of rules) {
      try {
        const parsed = parseJsonField(values[rule.name]);
        if (parsed === undefined) {
          continue;
        }
        if (rule.shape === "array" && !Array.isArray(parsed)) {
          setSubmitError(rule.name, `${rule.label} 必须是 JSON 数组。`);
          return false;
        }
        if (rule.shape === "object" && !isPlainJsonObject(parsed)) {
          setSubmitError(rule.name, `${rule.label} 必须是 JSON 对象。`);
          return false;
        }
      } catch {
        setSubmitError(rule.name, `${rule.label} 不是合法 JSON。`);
        return false;
      }
    }
    return true;
  }

  function setSubmitError(name: FormFieldName, message: string) {
    setActiveTab(tabForField(name));
    form.setError(name, { message });
    requestAnimationFrame(() => form.setFocus(name));
    toast.error(message);
  }

  function handleInvalid(errors: FieldErrors<FormValues>) {
    const [firstField] = Object.keys(errors);
    if (firstField) {
      const field = firstField as FormFieldName;
      setActiveTab(tabForField(field));
      requestAnimationFrame(() => form.setFocus(field));
    }
  }

  async function handleFormSubmit(event: FormEvent<HTMLFormElement>) {
    if (!advancedTouched) {
      await form.handleSubmit(handleSubmit, handleInvalid)(event);
      return;
    }

    event.preventDefault();
    form.clearErrors();
    const values = form.getValues();
    const result = advancedFormSchema.safeParse(values);
    if (!result.success) {
      for (const issue of result.error.issues) {
        const [field] = issue.path;
        if (typeof field === "string") {
          form.setError(field as FormFieldName, { message: issue.message });
        }
      }
      const firstIssue = result.error.issues[0];
      const [firstField] = firstIssue?.path || [];
      if (typeof firstField === "string") {
        setActiveTab(tabForField(firstField));
      }
      if (firstIssue?.message) toast.error(firstIssue.message);
      return;
    }

    await handleSubmit({ ...values, ...result.data });
  }

  function validateXrayConfig(values: FormValues, config: JsonObject) {
    form.clearErrors([
      "vless_decryption",
      "hysteria_version",
      "port",
      "out_address",
      "out_flow",
      "out_port",
      "reality_target",
      "reality_server_names",
      "reality_private_key",
      "reality_public_key",
      "reality_short_id",
      "reality_short_ids",
      "dns_servers_json",
      "vless_clients_json",
    ]);

    if (
      values.type === "inbound" &&
      !["tun", "wireguard"].includes(values.protocol || "") &&
      !isValidPort(config.port)
    ) {
      setSubmitError("port", "入站监听端口必填，范围为 1-65535。");
      return false;
    }

    if (
      values.type === "outbound" &&
      SERVER_OUTBOUND_PROTOCOLS.includes(values.protocol || "")
    ) {
      const settings = isJsonObject(config.settings) ? config.settings : {};
      if (!String(settings.address || "").trim()) {
        setSubmitError("out_address", "当前出站协议必须填写服务器地址。");
        return false;
      }
      if (!isValidPort(settings.port)) {
        setSubmitError("out_port", "当前出站协议必须填写 1-65535 的端口。");
        return false;
      }
    }

    if (values.type === "inbound" && values.protocol === "vless") {
      const settings = isJsonObject(config.settings) ? config.settings : {};
      const decryption = settings.decryption ?? values.vless_decryption;
      if (!String(decryption || "").trim()) {
        setSubmitError(
          "vless_decryption",
          "VLESS decryption 不能留空；禁用加密请填写 none。"
        );
        return false;
      }
      const clients = Array.isArray(settings.clients) ? settings.clients : [];
      if (
        clients.some(
          (client) =>
            isJsonObject(client) &&
            !isVlessFlowAllowed(values.network, client.flow)
        )
      ) {
        setSubmitError(
          "vless_clients_json",
          "VLESS Flow 仅适用于 RAW/TCP 传输；XHTTP、gRPC、WebSocket 等传输请留空。"
        );
        return false;
      }
    }

    const outboundSettings = isJsonObject(config.settings)
      ? config.settings
      : {};
    if (
      values.type === "outbound" &&
      values.protocol === "vless" &&
      !isVlessFlowAllowed(values.network, outboundSettings.flow)
    ) {
      setSubmitError(
        "out_flow",
        "VLESS Flow 仅适用于 RAW/TCP 传输；当前传输方式请留空。"
      );
      return false;
    }

    if (
      (values.type === "inbound" || values.type === "outbound") &&
      values.protocol === "hysteria"
    ) {
      const settings = isJsonObject(config.settings) ? config.settings : {};
      const version = settings.version ?? values.hysteria_version;
      if (Number(version) !== 2) {
        setSubmitError("hysteria_version", "Hysteria version 必须为 2。");
        return false;
      }
    }

    if (values.type === "dns") {
      const objectServers = safeJsonParse<unknown[]>(
        values.dns_servers_json || "",
        []
      );
      const hasQueryStrategyConflict = objectServers.some((server) => {
        if (!isJsonObject(server)) return false;
        if (values.query_strategy === "UseIPv4") {
          return server.queryStrategy === "UseIPv6";
        }
        if (values.query_strategy === "UseIPv6") {
          return server.queryStrategy === "UseIPv4";
        }
        return false;
      });
      if (hasQueryStrategyConflict) {
        setSubmitError(
          "dns_servers_json",
          "DNS Server Object 的 queryStrategy 与全局 Query Strategy 冲突，可能返回空响应。"
        );
        return false;
      }
    }

    const stream = isJsonObject(config.streamSettings)
      ? config.streamSettings
      : {};
    const reality = isJsonObject(stream.realitySettings)
      ? stream.realitySettings
      : {};
    if (values.type === "outbound" && stream.security === "reality") {
      if (!reality.password) {
        setSubmitError(
          "reality_public_key",
          "REALITY 出站必须填写服务端 Public Key / Password。"
        );
        return false;
      }
      if (!isRealityShortId(reality.shortId)) {
        setSubmitError(
          "reality_short_id",
          "REALITY Short ID 必须为偶数位十六进制，最长 16 位。"
        );
        return false;
      }
    }
    if (values.type !== "inbound" || stream.security !== "reality") {
      return true;
    }
    if (!(reality.target || reality.dest)) {
      setSubmitError(
        "reality_target",
        "REALITY 入站必须填写 Target，例如 ebay.com:443。缺少 target 时 xray-core 会把它当成客户端配置解析。"
      );
      return false;
    }
    if (!isRealityTarget(String(reality.target || reality.dest))) {
      setSubmitError(
        "reality_target",
        "REALITY Target 必须使用 host:port 格式，例如 example.com:443。"
      );
      return false;
    }
    const serverNames = Array.isArray(reality.serverNames)
      ? reality.serverNames
      : [];
    if (!serverNames.length) {
      setSubmitError(
        "reality_server_names",
        "REALITY 入站必须填写 Server Names，例如 ebay.com, www.ebay.com。"
      );
      return false;
    }
    if (!reality.privateKey) {
      setSubmitError(
        "reality_private_key",
        "REALITY 入站必须填写 Private Key。"
      );
      return false;
    }
    const shortIds = Array.isArray(reality.shortIds) ? reality.shortIds : [];
    if (!shortIds.length) {
      setSubmitError("reality_short_ids", "REALITY 入站必须填写 Short IDs。");
      return false;
    }
    const invalidShortId = shortIds.find(
      (shortId: unknown) => !isRealityShortId(shortId)
    );
    if (invalidShortId !== undefined) {
      setSubmitError(
        "reality_short_ids",
        "REALITY Short IDs 必须为十六进制，非空值需为偶数位且最长 16 位。"
      );
      return false;
    }
    return true;
  }

  async function handleSubmit(values: FormValues) {
    if (!validateJsonFields(values)) return;
    const sourceInitialValues = activeInitialValues || initialValues;
    const preserveInitialConfig =
      sourceInitialValues?.config &&
      !advancedTouched &&
      !hasStructuredConfigChanges(dirtyFields);
    const sourceConfig = preserveInitialConfig
      ? isJsonObject(sourceInitialValues.config)
        ? sourceInitialValues.config
        : {}
      : buildConfig(values);
    const config = normalizeVlessRealityGrpcConfig(sourceConfig, values.type);
    if (!validateXrayConfig(values, config)) return;
    const defaultVariables = safeJsonParse<JsonObject>(
      values.default_variables_json || "",
      {}
    );
    const streamSettings = isJsonObject(config.streamSettings)
      ? config.streamSettings
      : {};
    if (
      config.protocol === "vless" &&
      streamSettings.network === "grpc" &&
      streamSettings.security === "reality"
    ) {
      defaultVariables.flow = "none";
    }
    const conflicts = findDefaultVariableConflicts(
      values.type,
      config,
      defaultVariables
    );
    if (conflicts.length) {
      setSubmitError(
        "default_variables_json",
        `Default Variables 与模板配置存在冲突：${conflicts.join(", ")}。请移除重复默认值或保持一致。`
      );
      return;
    }
    const ok = await onSubmit({
      name: values.name,
      type: values.type,
      description: values.description,
      enabled: values.enabled,
      config: toAPIJsonObject(config),
      config_template: values.config_template,
      variables_schema: toAPIJsonObject(
        safeJsonParse(values.variables_schema_json || "", {})
      ),
      default_variables: toAPIJsonObject(defaultVariables),
      subscription_meta: toAPIJsonObject(
        safeJsonParse(values.subscription_meta_json || "", {})
      ),
    });
    if (ok) {
      setOpen(false);
    }
  }

  async function handleOpen() {
    setOpening(true);
    try {
      const values = loadInitialValues
        ? await loadInitialValues()
        : initialValues;
      if (loadInitialValues && !values) {
        return;
      }
      setActiveInitialValues(values);
      setActiveTab("form");
      setOpen(true);
    } finally {
      setOpening(false);
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen && loading) return;
    if (!nextOpen && form.formState.isDirty) {
      setDiscardOpen(true);
      return;
    }
    setOpen(nextOpen);
  }

  function discardChanges() {
    form.reset(form.getValues());
    setDiscardOpen(false);
    setOpen(false);
  }

  function renderTrigger() {
    if (isValidElement(trigger)) {
      const element = trigger as ReactElement<{
        disabled?: boolean;
        onClick?: () => void;
      }>;
      return cloneElement(element, {
        disabled: opening || element.props.disabled,
        onClick: handleOpen,
      });
    }
    return (
      <Button disabled={opening} onClick={handleOpen} type="button">
        {opening ? (
          <Icon className="mr-2 animate-spin" icon="mdi:loading" />
        ) : null}
        {trigger}
      </Button>
    );
  }

  function syncAdvancedJson() {
    const values = form.getValues();
    if (!validateJsonFields(values, false)) return;
    const config = buildConfig(values, false);
    form.setValue("advanced_json", formatJson(config), {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    });
    form.clearErrors("advanced_json");
    setAdvancedTouched(false);
  }

  const previewConfig = useMemo(() => {
    try {
      const values = {
        ...defaultValues(type),
        ...form.getValues(),
        ...watchedValues,
      } as FormValues;
      return {
        error: "",
        json: formatJson(buildConfig(values)),
      };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : "JSON preview error",
        json: "{}",
      };
    }
  }, [advancedTouched, form, type, watchedValues]);
  const typeLabel = String(t(`type.${type}`, type || ""));
  const summaryItems = useMemo(() => {
    const stateLabel = watchedValues?.enabled
      ? t("state.enabled", "Enabled")
      : t("state.disabled", "Disabled");
    const base = [
      { label: t("form.type", "Type"), value: typeLabel },
      { label: t("summary.status", "Status"), value: stateLabel },
    ];

    if (type === "routing") {
      return [
        ...base,
        {
          label: t("form.domainStrategy", "Domain Strategy"),
          value: watchedValues?.routing_domain_strategy,
        },
        {
          label: t("form.routingRules", "Routing Rules"),
          value: jsonArrayCount(watchedValues?.routing_rules_json),
        },
        {
          label: t("form.routingBalancers", "Balancers"),
          value: jsonArrayCount(watchedValues?.routing_balancers_json),
        },
      ];
    }

    if (type === "dns") {
      return [
        ...base,
        {
          label: t("form.dnsServers", "DNS Servers"),
          value:
            linesToArray(watchedValues?.servers).length +
            jsonArrayCount(watchedValues?.dns_servers_json),
        },
        {
          label: t("form.queryStrategy", "Query Strategy"),
          value: watchedValues?.query_strategy,
        },
      ];
    }

    if (type === "geodata") {
      return [
        ...base,
        {
          label: t("form.geodataAssets", "Geodata Assets"),
          value: jsonArrayCount(watchedValues?.geodata_assets_json),
        },
        {
          label: t("form.geodataCron", "Update Cron"),
          value: watchedValues?.geodata_cron,
        },
        {
          label: t("form.geodataOutbound", "Download Outbound"),
          value: watchedValues?.geodata_outbound,
        },
      ];
    }

    const protocolItems = [
      ...base,
      { label: t("form.protocol", "Protocol"), value: protocol },
      { label: t("form.tag", "Tag"), value: watchedValues?.tag },
    ];

    if (type === "inbound") {
      protocolItems.push(
        { label: t("form.listen", "Listen"), value: watchedValues?.listen },
        {
          label: t("form.port", "Port"),
          value: fieldString(watchedValues?.port),
        }
      );
    }

    if (supportsStreamSettings) {
      protocolItems.push(
        { label: t("form.network", "Network"), value: network },
        { label: t("form.security", "Security"), value: security }
      );
    }

    return protocolItems;
  }, [
    network,
    protocol,
    security,
    supportsStreamSettings,
    t,
    type,
    typeLabel,
    watchedValues,
  ]);

  return (
    <>
      <Sheet onOpenChange={handleOpenChange} open={open}>
        {renderTrigger()}
        <SheetContent className="xray-template-sheet w-[min(1240px,calc(100vw-24px))] max-w-full md:max-w-7xl">
          <SheetHeader className="xray-template-sheet-header">
            <SheetTitle className="xray-template-title">
              <span>{title}</span>
              <span className="xray-template-title-meta">
                <Badge variant="secondary">{typeLabel}</Badge>
                <Badge variant={advancedTouched ? "outline" : "secondary"}>
                  {advancedTouched ? "高级 JSON 生效" : "表单配置生效"}
                </Badge>
              </span>
            </SheetTitle>
            <SheetDescription>
              按 Xray 配置层级编辑协议、传输、安全与订阅映射；右侧会同步显示最终
              JSON 草稿。
            </SheetDescription>
          </SheetHeader>

          <div className="xray-template-layout grid h-[calc(100dvh-156px)] grid-cols-1 gap-0 overflow-hidden lg:grid-cols-[minmax(0,1fr)_280px]">
            <aside className="xray-template-summary min-h-0 space-y-4 overflow-y-auto">
              <div className="space-y-2">
                <div className="font-medium text-sm">配置概览</div>
                <div className="grid grid-cols-2 gap-2">
                  {summaryItems.map((item) => (
                    <SummaryPill
                      key={item.label}
                      label={item.label}
                      value={item.value}
                    />
                  ))}
                </div>
              </div>
              <div className="xray-template-preview space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <div className="font-medium text-sm">草稿预览</div>
                  <Badge
                    variant={previewConfig.error ? "outline" : "secondary"}
                  >
                    {previewConfig.error ? "异常" : "JSON"}
                  </Badge>
                </div>
                {previewConfig.error ? (
                  <Alert variant="destructive">
                    <CircleAlert />
                    <AlertDescription>{previewConfig.error}</AlertDescription>
                  </Alert>
                ) : null}
                <Textarea
                  className="max-h-[340px] min-h-[260px] resize-none font-mono text-[11px]"
                  readOnly
                  value={previewConfig.json}
                />
              </div>
            </aside>

            <ScrollArea className="xray-template-form-pane min-h-0">
              <Form {...form}>
                <AdminForm
                  className="space-y-5 py-4"
                  id="xray-template-form"
                  onSubmit={handleFormSubmit}
                >
                  <section className="xray-form-basics space-y-4">
                    <div>
                      <div className="font-medium text-sm">基础信息</div>
                      <div className="text-muted-foreground text-xs">
                        先确定模板类型和命名，再进入具体协议参数。
                      </div>
                    </div>
                    <div className="xray-template-basics-grid">
                      <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("form.name", "Name")}
                              <span
                                aria-hidden="true"
                                className="ml-1 text-destructive"
                              >
                                *
                              </span>
                            </FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "form.namePlaceholder",
                                  "Template name"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="type"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.type", "Type")}</FormLabel>
                            <Select
                              disabled={Boolean(initialValues?.id)}
                              onValueChange={(value) => {
                                const next = value as XrayTemplateType;
                                setAdvancedTouched(false);
                                form.reset({
                                  ...defaultValues(next),
                                  name: form.getValues("name"),
                                  type: next,
                                  description: form.getValues("description"),
                                  enabled: form.getValues("enabled"),
                                });
                              }}
                              value={field.value}
                            >
                              <FormControl>
                                <SelectTrigger className="w-full">
                                  <SelectValue />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                {XRAY_TEMPLATE_TYPES.map((item) => (
                                  <SelectItem key={item} value={item}>
                                    {t(`type.${item}`, item)}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="description"
                        render={({ field }) => (
                          <FormItem className="xray-template-description-field">
                            <FormLabel>
                              {t("form.description", "Description")}
                            </FormLabel>
                            <FormControl>
                              <Textarea
                                className="min-h-20 resize-y"
                                onChange={field.onChange}
                                placeholder={t(
                                  "form.descriptionPlaceholder",
                                  "Optional note"
                                )}
                                value={field.value || ""}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <SwitchField
                        control={form.control}
                        label={t("form.enabled", "Enabled")}
                        name="enabled"
                      />
                    </div>
                  </section>

                  <details className="xray-mobile-preview lg:hidden">
                    <summary>
                      <span>
                        {t("form.mobilePreview", "Configuration preview")}
                      </span>
                      <Badge
                        variant={previewConfig.error ? "outline" : "secondary"}
                      >
                        {previewConfig.error
                          ? t("form.previewInvalid", "Invalid")
                          : "JSON"}
                      </Badge>
                    </summary>
                    {previewConfig.error ? (
                      <Alert className="mt-3" variant="destructive">
                        <CircleAlert />
                        <AlertDescription>
                          {previewConfig.error}
                        </AlertDescription>
                      </Alert>
                    ) : null}
                    <Textarea
                      className="mt-3 max-h-72 min-h-48 resize-y font-mono text-[11px]"
                      readOnly
                      value={previewConfig.json}
                    />
                  </details>

                  <Tabs
                    className="xray-template-tabs"
                    onValueChange={(value) =>
                      setActiveTab(value as XrayFormTab)
                    }
                    value={activeTab}
                  >
                    <TabsList className="grid w-full grid-cols-4 md:w-[520px]">
                      <TabsTrigger
                        className="min-w-0 px-1.5 sm:px-3"
                        title={t("tabs.form", "Form")}
                        value="form"
                      >
                        <SlidersHorizontal
                          aria-hidden="true"
                          className="size-4 shrink-0"
                        />
                        <span className="hidden truncate sm:inline">
                          {t("tabs.form", "Form")}
                        </span>
                        <TabErrorCount count={tabErrorCounts.form} />
                      </TabsTrigger>
                      <TabsTrigger
                        className="min-w-0 px-1.5 sm:px-3"
                        title={t("tabs.advanced", "Advanced JSON")}
                        value="advanced"
                      >
                        <Braces
                          aria-hidden="true"
                          className="size-4 shrink-0"
                        />
                        <span className="hidden truncate sm:inline">
                          {t("tabs.advanced", "Advanced JSON")}
                        </span>
                        <TabErrorCount count={tabErrorCounts.advanced} />
                      </TabsTrigger>
                      <TabsTrigger
                        className="min-w-0 px-1.5 sm:px-3"
                        title={t("tabs.template", "Template")}
                        value="template"
                      >
                        <FileCode2
                          aria-hidden="true"
                          className="size-4 shrink-0"
                        />
                        <span className="hidden truncate sm:inline">
                          {t("tabs.template", "Template")}
                        </span>
                        <TabErrorCount count={tabErrorCounts.template} />
                      </TabsTrigger>
                      <TabsTrigger
                        className="min-w-0 px-1.5 sm:px-3"
                        title={t("tabs.subscription", "Subscription")}
                        value="subscription"
                      >
                        <Share2
                          aria-hidden="true"
                          className="size-4 shrink-0"
                        />
                        <span className="hidden truncate sm:inline">
                          {t("tabs.subscription", "Subscription")}
                        </span>
                        <TabErrorCount count={tabErrorCounts.subscription} />
                      </TabsTrigger>
                    </TabsList>

                    <TabsContent
                      className="xray-form-tab space-y-5 pt-4"
                      value="form"
                    >
                      {type === "routing" ? (
                        <div className="space-y-4">
                          <FormSectionHeading
                            description="先选择常用策略，再按实际出口与入站标签调整规则顺序。"
                            title="路由策略"
                          />
                          <div className="space-y-2">
                            <div className="font-medium text-sm">
                              {t("form.routingPresets", "Routing Presets")}
                            </div>
                            <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
                              {ROUTING_PRESETS.map((preset) => (
                                <Button
                                  className="h-auto justify-start whitespace-normal p-3 text-left"
                                  key={preset.id}
                                  onClick={() => applyRoutingPreset(preset)}
                                  type="button"
                                  variant="outline"
                                >
                                  <span className="space-y-1">
                                    <span className="block font-medium">
                                      {preset.label}
                                    </span>
                                    <span className="block text-muted-foreground text-xs leading-5">
                                      {preset.description}
                                    </span>
                                  </span>
                                </Button>
                              ))}
                            </div>
                          </div>
                          <FormField
                            control={form.control}
                            name="routing_domain_strategy"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  <XrayFieldLabel fieldKey="routing_domain_strategy">
                                    {t(
                                      "form.domainStrategy",
                                      "Domain Strategy"
                                    )}
                                  </XrayFieldLabel>
                                </FormLabel>
                                <XrayFieldDescription fieldKey="routing_domain_strategy" />
                                <Select
                                  onValueChange={field.onChange}
                                  value={field.value}
                                >
                                  <FormControl>
                                    <SelectTrigger className="w-full">
                                      <SelectValue />
                                    </SelectTrigger>
                                  </FormControl>
                                  <SelectContent>
                                    {ROUTING_DOMAIN_STRATEGIES.map((item) => (
                                      <SelectItem key={item} value={item}>
                                        {item}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <div className="space-y-4">
                            <JsonArrayObjectField
                              addLabel="添加路由规则"
                              columns={ROUTING_RULE_COLUMNS}
                              control={form.control}
                              defaultItem={{
                                outboundTag: "direct",
                                type: "field",
                              }}
                              description={t(
                                "form.routingRulesDesc",
                                "按行配置 RuleObject，可用模板引用填入 inboundTag/outboundTag。"
                              )}
                              label={t("form.routingRules", "Routing Rules")}
                              name="routing_rules_json"
                            />
                            <JsonArrayObjectField
                              addLabel="添加负载均衡器"
                              columns={ROUTING_BALANCER_COLUMNS}
                              control={form.control}
                              description={t(
                                "form.routingBalancersDesc",
                                "按行配置 BalancerObject。"
                              )}
                              label={t("form.routingBalancers", "Balancers")}
                              name="routing_balancers_json"
                            />
                          </div>
                        </div>
                      ) : type === "geodata" ? (
                        <div className="space-y-4">
                          <FormSectionHeading
                            description="设置数据更新时间、下载出口与需要维护的资源文件。"
                            title="更新计划"
                          />
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <FormField
                              control={form.control}
                              name="geodata_cron"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    {t("form.geodataCron", "Update Cron")}
                                  </FormLabel>
                                  <FormDescription>
                                    {t(
                                      "form.geodataCronDesc",
                                      "Five-field cron expression in the Xray runtime local timezone, e.g. 0 4 * * *."
                                    )}
                                  </FormDescription>
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            <FormField
                              control={form.control}
                              name="geodata_outbound"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    {t(
                                      "form.geodataOutbound",
                                      "Download Outbound"
                                    )}
                                  </FormLabel>
                                  <FormDescription>
                                    {t(
                                      "form.geodataOutboundDesc",
                                      "Optional outbound tag used when downloading geodata files."
                                    )}
                                  </FormDescription>
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      placeholder="direct"
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                          </div>
                          <JsonArrayObjectField
                            addLabel="添加文件"
                            columns={GEODATA_ASSET_COLUMNS}
                            control={form.control}
                            description={t(
                              "form.geodataAssetsDesc",
                              "配置要下载到 Xray asset 目录的地理数据文件。"
                            )}
                            label={t("form.geodataAssets", "Geodata Assets")}
                            name="geodata_assets_json"
                          />
                        </div>
                      ) : type !== "dns" ? (
                        <>
                          <FormSectionHeading
                            description="确定协议身份、监听入口以及 streamSettings 的基础组合。"
                            title="协议与监听"
                          />
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                            <FormField
                              control={form.control}
                              name="protocol"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="protocol">
                                      {t("form.protocol", "Protocol")}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="protocol" />
                                  <Select
                                    onValueChange={field.onChange}
                                    value={field.value}
                                  >
                                    <FormControl>
                                      <SelectTrigger className="w-full">
                                        <SelectValue />
                                      </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                      {protocolOptions.map((item) => (
                                        <SelectItem key={item} value={item}>
                                          {item}
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            <FormField
                              control={form.control}
                              name="tag"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="tag">
                                      {t("form.tag", "Tag")}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="tag" />
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      placeholder="proxy-in"
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            {type === "inbound" ? (
                              <FormField
                                control={form.control}
                                name="port"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>
                                      <XrayFieldLabel fieldKey="port">
                                        {t("form.port", "Port")}
                                      </XrayFieldLabel>
                                      <span
                                        aria-hidden="true"
                                        className="ml-1 text-destructive"
                                      >
                                        *
                                      </span>
                                    </FormLabel>
                                    <XrayFieldDescription fieldKey="port" />
                                    <FormControl>
                                      <EnhancedInput
                                        max={65_535}
                                        min={1}
                                        onValueChange={field.onChange}
                                        placeholder="443"
                                        step={1}
                                        type="number"
                                        value={field.value}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            ) : null}
                            {type === "inbound" ? (
                              <FormField
                                control={form.control}
                                name="listen"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>
                                      <XrayFieldLabel fieldKey="listen">
                                        {t("form.listen", "Listen")}
                                      </XrayFieldLabel>
                                    </FormLabel>
                                    <XrayFieldDescription fieldKey="listen" />
                                    <FormControl>
                                      <EnhancedInput
                                        onValueChange={field.onChange}
                                        placeholder="0.0.0.0"
                                        value={field.value || ""}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            ) : null}
                            {supportsStreamSettings ? (
                              <>
                                <FormField
                                  control={form.control}
                                  name="network"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="network">
                                          {t("form.network", "Network")}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="network" />
                                      <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                      >
                                        <FormControl>
                                          <SelectTrigger className="w-full">
                                            <SelectValue />
                                          </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                          {networkOptions.map((item) => (
                                            <SelectItem key={item} value={item}>
                                              {item}
                                            </SelectItem>
                                          ))}
                                        </SelectContent>
                                      </Select>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                <FormField
                                  control={form.control}
                                  name="security"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="security">
                                          {t("form.security", "Security")}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription
                                        description={
                                          network === "hysteria"
                                            ? t(
                                                "form.hysteriaTlsRequired",
                                                "Hysteria transport requires TLS; the security layer is fixed to tls."
                                              )
                                            : undefined
                                        }
                                        fieldKey="security"
                                      />
                                      <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                      >
                                        <FormControl>
                                          <SelectTrigger className="w-full">
                                            <SelectValue />
                                          </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                          {securityOptions.map((item) => (
                                            <SelectItem key={item} value={item}>
                                              {item}
                                            </SelectItem>
                                          ))}
                                        </SelectContent>
                                      </Select>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </>
                            ) : null}
                          </div>

                          {supportsStreamSettings ? (
                            <FormSectionHeading
                              description="只展示当前传输方式会写入的字段，切换传输后请复核路径与高级参数。"
                              title="传输参数"
                            />
                          ) : null}

                          {supportsStreamSettings &&
                          ["websocket", "xhttp", "httpupgrade"].includes(
                            network || ""
                          ) ? (
                            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                              <FormField
                                control={form.control}
                                name="host"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>
                                      <XrayFieldLabel fieldKey="host">
                                        {t("form.host", "Host")}
                                      </XrayFieldLabel>
                                    </FormLabel>
                                    <XrayFieldDescription fieldKey="host" />
                                    <FormControl>
                                      <EnhancedInput
                                        onValueChange={field.onChange}
                                        placeholder="example.com"
                                        value={field.value || ""}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                              <FormField
                                control={form.control}
                                name="path"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>
                                      <XrayFieldLabel fieldKey="path">
                                        {t("form.path", "Path")}
                                      </XrayFieldLabel>
                                    </FormLabel>
                                    <XrayFieldDescription fieldKey="path" />
                                    <FormControl>
                                      <EnhancedInput
                                        onValueChange={field.onChange}
                                        placeholder="/ray"
                                        value={field.value || ""}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            </div>
                          ) : null}

                          {supportsStreamSettings && network === "xhttp" ? (
                            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                              <SelectField
                                control={form.control}
                                label={t("form.xhttpMode", "XHTTP Mode")}
                                name="xhttp_mode"
                                options={XHTTP_MODES}
                              />
                              <JsonObjectField
                                className="md:col-span-2"
                                control={form.control}
                                description="XHTTP extra 对象。需要 xmux.maxConnections 这类嵌套字段时，值类型选择 JSON。"
                                label={t("form.xhttpExtra", "XHTTP Extra")}
                                name="xhttp_extra_json"
                              />
                            </div>
                          ) : null}

                          {supportsStreamSettings && network === "grpc" ? (
                            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                              <InputField
                                control={form.control}
                                label={t("form.serviceName", "Service Name")}
                                name="service_name"
                              />
                              <InputField
                                control={form.control}
                                label={t("form.grpcAuthority", "Authority")}
                                name="grpc_authority"
                              />
                              {type === "outbound" ? (
                                <>
                                  <InputField
                                    control={form.control}
                                    label={t(
                                      "form.grpcUserAgent",
                                      "User-Agent"
                                    )}
                                    name="grpc_user_agent"
                                    placeholder="alpha-client"
                                  />
                                  <SwitchField
                                    control={form.control}
                                    label={t(
                                      "form.grpcMultiMode",
                                      "Multi Mode"
                                    )}
                                    name="grpc_multi_mode"
                                  />
                                  <InputField
                                    control={form.control}
                                    label={t(
                                      "form.grpcIdleTimeout",
                                      "Idle Timeout"
                                    )}
                                    name="grpc_idle_timeout"
                                    type="number"
                                  />
                                  <InputField
                                    control={form.control}
                                    label={t(
                                      "form.grpcHealthCheckTimeout",
                                      "Health Check Timeout"
                                    )}
                                    name="grpc_health_check_timeout"
                                    type="number"
                                  />
                                  <SwitchField
                                    control={form.control}
                                    label={t(
                                      "form.grpcPermitWithoutStream",
                                      "Permit Without Stream"
                                    )}
                                    name="grpc_permit_without_stream"
                                  />
                                  <InputField
                                    control={form.control}
                                    label={t(
                                      "form.grpcInitialWindowsSize",
                                      "Initial Window Size"
                                    )}
                                    name="grpc_initial_windows_size"
                                    type="number"
                                  />
                                </>
                              ) : null}
                            </div>
                          ) : null}

                          {supportsStreamSettings && network === "mkcp" ? (
                            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                              <InputField
                                control={form.control}
                                label="MTU"
                                name="kcp_mtu"
                                type="number"
                              />
                              <InputField
                                control={form.control}
                                label="TTI"
                                name="kcp_tti"
                                type="number"
                              />
                              <InputField
                                control={form.control}
                                label="Uplink Capacity"
                                name="kcp_uplink_capacity"
                                type="number"
                              />
                              <InputField
                                control={form.control}
                                label="Downlink Capacity"
                                name="kcp_downlink_capacity"
                                type="number"
                              />
                              <SwitchField
                                control={form.control}
                                label="Congestion"
                                name="kcp_congestion"
                              />
                              <InputField
                                control={form.control}
                                label="Header Type"
                                name="kcp_header_type"
                                placeholder="none / srtp / utp / wechat-video"
                              />
                            </div>
                          ) : null}

                          {supportsStreamSettings && network === "hysteria" ? (
                            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                              <InputField
                                control={form.control}
                                description="Hysteria 传输层认证；入站存在 users 时会被 users 覆盖。"
                                label="Auth"
                                name="hysteria_auth"
                                placeholder="{{ .Vars.password }}"
                              />
                              <InputField
                                control={form.control}
                                description="QUIC UDP 空闲等待时间，单位秒；官方默认 60。"
                                label="UDP Idle Timeout"
                                name="udp_idle_timeout"
                                type="number"
                              />
                              <JsonObjectField
                                className="md:col-span-2"
                                control={form.control}
                                description="Hysteria HTTP/3 伪装对象；支持 string、file、proxy 三种 type。"
                                label="Masquerade"
                                name="hysteria_masquerade_json"
                              />
                            </div>
                          ) : null}

                          {supportsStreamSettings &&
                          ["raw", "mkcp", "hysteria"].includes(
                            network || ""
                          ) ? (
                            <FormField
                              control={form.control}
                              name={
                                network === "mkcp"
                                  ? "kcp_settings_json"
                                  : network === "hysteria"
                                    ? "hysteria_settings_json"
                                    : "raw_settings_json"
                              }
                              render={({ field }) => (
                                <FormItem>
                                  <CompositeFieldLabel>
                                    <XrayFieldLabel
                                      fieldKey={
                                        network === "mkcp"
                                          ? "kcp_settings_json"
                                          : network === "hysteria"
                                            ? "hysteria_settings_json"
                                            : "raw_settings_json"
                                      }
                                    >
                                      {network === "mkcp"
                                        ? t("form.kcpSettings", "KCP Settings")
                                        : network === "hysteria"
                                          ? t(
                                              "form.hysteriaSettings",
                                              "Hysteria Settings"
                                            )
                                          : t(
                                              "form.rawSettings",
                                              "Raw Settings"
                                            )}
                                    </XrayFieldLabel>
                                  </CompositeFieldLabel>
                                  <XrayFieldDescription
                                    description={
                                      network === "mkcp"
                                        ? "KcpObject advanced fields. Form values above are merged first, this JSON can override or add rare fields."
                                        : network === "hysteria"
                                          ? "HysteriaObject for streamSettings.hysteriaSettings."
                                          : "RawObject for streamSettings.rawSettings."
                                    }
                                    fieldKey={
                                      network === "mkcp"
                                        ? "kcp_settings_json"
                                        : network === "hysteria"
                                          ? "hysteria_settings_json"
                                          : "raw_settings_json"
                                    }
                                  />
                                  <JsonObjectEditor
                                    onChange={field.onChange}
                                    value={field.value}
                                  />
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                          ) : null}

                          {supportsStreamSettings &&
                          (security === "tls" ||
                            (security === "reality" && type === "outbound")) ? (
                            <>
                              <FormSectionHeading
                                description="配置证书校验、SNI、指纹与 REALITY 身份参数。"
                                title="传输安全"
                              />
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <FormField
                                  control={form.control}
                                  name="sni"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="sni">
                                          {t("form.sni", "SNI")}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="sni" />
                                      <FormControl>
                                        <EnhancedInput
                                          onValueChange={field.onChange}
                                          placeholder="example.com"
                                          value={field.value || ""}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                {security === "tls" ||
                                (security === "reality" &&
                                  type === "outbound") ? (
                                  <FormField
                                    control={form.control}
                                    name="fingerprint"
                                    render={({ field }) => (
                                      <FormItem>
                                        <FormLabel>
                                          <XrayFieldLabel fieldKey="fingerprint">
                                            {t(
                                              "form.fingerprint",
                                              "Fingerprint"
                                            )}
                                          </XrayFieldLabel>
                                        </FormLabel>
                                        <XrayFieldDescription fieldKey="fingerprint" />
                                        <FormControl>
                                          <EnhancedInput
                                            onValueChange={field.onChange}
                                            placeholder="chrome"
                                            value={field.value || ""}
                                          />
                                        </FormControl>
                                        <FormMessage />
                                      </FormItem>
                                    )}
                                  />
                                ) : null}
                                {security === "tls" && type === "outbound" ? (
                                  <>
                                    <FormField
                                      control={form.control}
                                      name="pinned_peer_cert_sha256"
                                      render={({ field }) => (
                                        <FormItem>
                                          <FormLabel>
                                            <XrayFieldLabel fieldKey="pinned_peer_cert_sha256">
                                              {t(
                                                "form.pinnedPeerCertSha256",
                                                "Pinned Cert SHA-256"
                                              )}
                                            </XrayFieldLabel>
                                          </FormLabel>
                                          <XrayFieldDescription fieldKey="pinned_peer_cert_sha256" />
                                          <FormControl>
                                            <EnhancedInput
                                              onValueChange={field.onChange}
                                              placeholder="AA:BB:CC:..."
                                              value={field.value || ""}
                                            />
                                          </FormControl>
                                          <FormMessage />
                                        </FormItem>
                                      )}
                                    />
                                    <FormField
                                      control={form.control}
                                      name="verify_peer_cert_by_name"
                                      render={({ field }) => (
                                        <FormItem>
                                          <FormLabel>
                                            <XrayFieldLabel fieldKey="verify_peer_cert_by_name">
                                              {t(
                                                "form.verifyPeerCertByName",
                                                "Verify Cert Name"
                                              )}
                                            </XrayFieldLabel>
                                          </FormLabel>
                                          <XrayFieldDescription fieldKey="verify_peer_cert_by_name" />
                                          <FormControl>
                                            <EnhancedInput
                                              onValueChange={field.onChange}
                                              placeholder="example.com"
                                              value={field.value || ""}
                                            />
                                          </FormControl>
                                          <FormMessage />
                                        </FormItem>
                                      )}
                                    />
                                  </>
                                ) : null}
                              </div>
                            </>
                          ) : null}

                          {supportsStreamSettings && security === "reality" ? (
                            <>
                              {type === "inbound" ? (
                                <FormSectionHeading
                                  description="Target 使用 host:port；Short ID 为偶数位十六进制且最长 16 位。"
                                  title="REALITY 服务端"
                                />
                              ) : null}
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <div className="flex justify-end gap-2 md:col-span-2">
                                  {type === "inbound" ? (
                                    <Button
                                      onClick={() => {
                                        const pair = generateRealityKeyPair();
                                        form.setValue(
                                          "reality_private_key",
                                          pair.privateKey
                                        );
                                        form.setValue(
                                          "reality_public_key",
                                          pair.publicKey
                                        );
                                      }}
                                      size="sm"
                                      type="button"
                                      variant="outline"
                                    >
                                      {t(
                                        "form.generateRealityKey",
                                        "Generate Key"
                                      )}
                                    </Button>
                                  ) : null}
                                  <Button
                                    onClick={() => {
                                      const next = generateRealityShortId();
                                      if (type === "outbound") {
                                        form.setValue("reality_short_id", next);
                                      } else {
                                        const current =
                                          form.getValues("reality_short_ids") ||
                                          "";
                                        form.setValue(
                                          "reality_short_ids",
                                          current ? `${current}, ${next}` : next
                                        );
                                      }
                                    }}
                                    size="sm"
                                    type="button"
                                    variant="outline"
                                  >
                                    {t(
                                      "form.generateShortId",
                                      "Generate Short ID"
                                    )}
                                  </Button>
                                </div>
                                {type === "inbound" ? (
                                  <>
                                    <SwitchField
                                      control={form.control}
                                      label="Show"
                                      name="reality_show"
                                    />
                                    <InputField
                                      control={form.control}
                                      description="必填。REALITY 服务端目标站点，当前 Xray 字段是 target；例如 ebay.com:443。缺少时会被 xray-core 当成客户端 REALITY 配置解析。"
                                      label="Target"
                                      name="reality_target"
                                      placeholder="example.com:443"
                                      required
                                    />
                                    <InputField
                                      control={form.control}
                                      description="必填。REALITY 服务端允许的 SNI 列表，多个值用英文逗号分隔。"
                                      label="Server Names"
                                      name="reality_server_names"
                                      placeholder="example.com, www.example.com"
                                      required
                                    />
                                    <InputField
                                      control={form.control}
                                      label="XVer"
                                      name="reality_xver"
                                      type="number"
                                    />
                                    <InputField
                                      control={form.control}
                                      label={t(
                                        "form.realityShortIds",
                                        "Short IDs"
                                      )}
                                      name="reality_short_ids"
                                      placeholder="comma separated"
                                      required
                                    />
                                    <InputField
                                      control={form.control}
                                      label={t(
                                        "form.realityPrivateKey",
                                        "Private Key"
                                      )}
                                      name="reality_private_key"
                                      required
                                    />
                                    <InputField
                                      control={form.control}
                                      label="Min Client Ver"
                                      name="reality_min_client_ver"
                                    />
                                    <InputField
                                      control={form.control}
                                      label="Max Client Ver"
                                      name="reality_max_client_ver"
                                    />
                                    <InputField
                                      control={form.control}
                                      label="Max Time Diff"
                                      name="reality_max_time_diff"
                                      type="number"
                                    />
                                    <InputField
                                      control={form.control}
                                      description="可选。使用 xray mldsa65 生成的服务端私有种子，用于为临时证书增加 ML-DSA-65 签名。"
                                      label="ML-DSA-65 Seed"
                                      name="reality_mldsa65_seed"
                                    />
                                  </>
                                ) : (
                                  <>
                                    <InputField
                                      control={form.control}
                                      description={t(
                                        "form.realityPublicKeyDescription",
                                        "Reality client uses password for the server public key."
                                      )}
                                      label="Public Key / Password"
                                      name="reality_public_key"
                                      required
                                    />
                                    <InputField
                                      control={form.control}
                                      label="Short ID"
                                      name="reality_short_id"
                                    />
                                    <InputField
                                      control={form.control}
                                      label="Spider X"
                                      name="reality_spider_x"
                                      placeholder="/"
                                    />
                                    <InputField
                                      control={form.control}
                                      description="可选。与服务端 ML-DSA-65 Seed 对应的客户端校验公钥。"
                                      label="ML-DSA-65 Verify"
                                      name="reality_mldsa65_verify"
                                    />
                                  </>
                                )}
                              </div>
                            </>
                          ) : null}

                          {supportsStreamSettings ? (
                            <>
                              <FormSectionHeading
                                description="仅在运行环境明确支持时配置 Sockopt 与 Final Mask。"
                                title="网络高级选项"
                              />
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <FormField
                                  control={form.control}
                                  name="sockopt_json"
                                  render={({ field }) => (
                                    <FormItem>
                                      <CompositeFieldLabel>
                                        <XrayFieldLabel fieldKey="sockopt_json">
                                          {t("form.sockopt", "Sockopt")}
                                        </XrayFieldLabel>
                                      </CompositeFieldLabel>
                                      <XrayFieldDescription
                                        description={
                                          <>
                                            SockoptObject JSON, e.g.{" "}
                                            {
                                              '{"mark":255,"tproxy":"tproxy","dialerProxy":"proxy"}'
                                            }
                                          </>
                                        }
                                        fieldKey="sockopt_json"
                                      />
                                      <JsonObjectEditor
                                        onChange={field.onChange}
                                        value={field.value}
                                      />
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                <FormField
                                  control={form.control}
                                  name="finalmask_json"
                                  render={({ field }) => (
                                    <FormItem>
                                      <CompositeFieldLabel>
                                        <XrayFieldLabel fieldKey="finalmask_json">
                                          {t("form.finalmask", "Final Mask")}
                                        </XrayFieldLabel>
                                      </CompositeFieldLabel>
                                      <XrayFieldDescription
                                        description="FinalMaskObject JSON with tcp/udp/quicParams fields."
                                        fieldKey="finalmask_json"
                                      />
                                      <JsonObjectEditor
                                        onChange={field.onChange}
                                        value={field.value}
                                      />
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </div>
                            </>
                          ) : null}

                          {type === "inbound" ? (
                            <>
                              <FormSectionHeading
                                description="控制目标识别、路由覆盖范围以及明确排除的域名。"
                                title="流量探测"
                              />
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <SwitchField
                                  control={form.control}
                                  label={t("form.sniffing", "Sniffing")}
                                  name="sniffing"
                                />
                                <SwitchField
                                  control={form.control}
                                  label={t(
                                    "form.sniffingMetadataOnly",
                                    "Metadata Only"
                                  )}
                                  name="sniffing_metadata_only"
                                />
                                <SwitchField
                                  control={form.control}
                                  label={t(
                                    "form.sniffingRouteOnly",
                                    "Route Only"
                                  )}
                                  name="sniffing_route_only"
                                />
                                <FormField
                                  control={form.control}
                                  name="dest_override"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="dest_override">
                                          {t(
                                            "form.destOverride",
                                            "Dest Override"
                                          )}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="dest_override" />
                                      <FormControl>
                                        <EnhancedInput
                                          onValueChange={field.onChange}
                                          placeholder="http, tls, quic"
                                          value={field.value || ""}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                <FormField
                                  control={form.control}
                                  name="sniffing_domains_excluded"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="sniffing_domains_excluded">
                                          {t(
                                            "form.sniffingDomainsExcluded",
                                            "Domains Excluded"
                                          )}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="sniffing_domains_excluded" />
                                      <FormControl>
                                        <EnhancedInput
                                          onValueChange={field.onChange}
                                          placeholder="example.com, geosite:cn"
                                          value={field.value || ""}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </div>
                            </>
                          ) : null}

                          <FormSectionHeading
                            description="编辑当前协议专属字段；Settings JSON 仅用于补充少见覆盖项。"
                            title="协议设置"
                          />

                          {type === "outbound" && protocol === "freedom" ? (
                            <div className="space-y-4">
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <FormField
                                  control={form.control}
                                  name="domain_strategy"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="domain_strategy">
                                          {t(
                                            "form.domainStrategy",
                                            "Domain Strategy"
                                          )}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="domain_strategy" />
                                      <FormControl>
                                        <EnhancedInput
                                          onValueChange={field.onChange}
                                          placeholder="AsIs / UseIP / UseIPv4 / UseIPv6"
                                          value={field.value || ""}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                <FormField
                                  control={form.control}
                                  name="redirect"
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormLabel>
                                        <XrayFieldLabel fieldKey="redirect">
                                          {t("form.redirect", "Redirect")}
                                        </XrayFieldLabel>
                                      </FormLabel>
                                      <XrayFieldDescription fieldKey="redirect" />
                                      <FormControl>
                                        <EnhancedInput
                                          onValueChange={field.onChange}
                                          placeholder="127.0.0.1:3366"
                                          value={field.value || ""}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                <InputField
                                  control={form.control}
                                  label="User Level"
                                  name="user_level"
                                  type="number"
                                />
                                <InputField
                                  control={form.control}
                                  label="Proxy Protocol"
                                  name="proxy_protocol"
                                  type="number"
                                />
                              </div>
                              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <JsonObjectField
                                  control={form.control}
                                  description='Freedom fragment object: {"packets":"tlshello","length":"100-200","interval":"10-20"}'
                                  label="Fragment"
                                  name="fragment_json"
                                />
                                <JsonArrayObjectField
                                  addLabel="添加噪声"
                                  columns={FREEDOM_NOISE_COLUMNS}
                                  control={form.control}
                                  description="Freedom noises 列表。"
                                  label="Noises"
                                  name="noises_json"
                                />
                              </div>
                            </div>
                          ) : null}

                          {type === "outbound" && protocol === "blackhole" ? (
                            <FormField
                              control={form.control}
                              name="response_type"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="response_type">
                                      {t("form.responseType", "Response Type")}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="response_type" />
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      placeholder="http"
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                          ) : null}

                          <ProtocolSettingsFields
                            control={form.control}
                            key={`${type}:${protocol}`}
                            protocol={protocol}
                            type={type}
                          />

                          <FormField
                            control={form.control}
                            name="settings_json"
                            render={({ field }) => (
                              <FormItem>
                                <CompositeFieldLabel>
                                  <XrayFieldLabel fieldKey="settings_json">
                                    {t("form.settingsJson", "Settings JSON")}
                                  </XrayFieldLabel>
                                </CompositeFieldLabel>
                                <XrayFieldDescription
                                  description={t(
                                    "form.settingsJsonDesc",
                                    "Protocol-specific Xray settings. These fields are kept as JSON so advanced protocol details are not lost."
                                  )}
                                  fieldKey="settings_json"
                                />
                                <JsonObjectEditor
                                  addLabel="添加覆盖字段"
                                  emptyText="没有高级覆盖字段。"
                                  onChange={field.onChange}
                                  value={field.value}
                                />
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </>
                      ) : (
                        <>
                          <FormSectionHeading
                            description="配置全局解析策略、服务器来源、缓存与回退行为。"
                            title="DNS 解析"
                          />
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <FormField
                              control={form.control}
                              name="tag"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="tag">
                                      {t("form.tag", "Tag")}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="tag" />
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      placeholder="dns"
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            <FormField
                              control={form.control}
                              name="client_ip"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="client_ip">
                                      {t("form.clientIp", "Client IP")}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="client_ip" />
                                  <FormControl>
                                    <EnhancedInput
                                      onValueChange={field.onChange}
                                      placeholder="1.2.3.4"
                                      value={field.value || ""}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                          </div>
                          <FormField
                            control={form.control}
                            name="servers"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  <XrayFieldLabel fieldKey="servers">
                                    {t("form.dnsServers", "DNS Servers")}
                                  </XrayFieldLabel>
                                </FormLabel>
                                <XrayFieldDescription
                                  description={t(
                                    "form.dnsServersDesc",
                                    "One server per line. Plain strings are supported, e.g. 1.1.1.1, https://dns.google/dns-query."
                                  )}
                                  fieldKey="servers"
                                />
                                <FormControl>
                                  <Textarea
                                    className="min-h-28"
                                    onChange={field.onChange}
                                    value={field.value || ""}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <JsonArrayObjectField
                            addLabel="添加 DNS 对象"
                            columns={DNS_SERVER_OBJECT_COLUMNS}
                            control={form.control}
                            description={t(
                              "form.dnsServerObjectsDesc",
                              "需要 domains、expectedIPs、skipFallback 等条件时，用表单新增一个 DNS Server Object。"
                            )}
                            label={t(
                              "form.dnsServerObjects",
                              "DNS Server Objects"
                            )}
                            name="dns_servers_json"
                          />
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <FormField
                              control={form.control}
                              name="query_strategy"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="query_strategy">
                                      {t(
                                        "form.queryStrategy",
                                        "Query Strategy"
                                      )}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="query_strategy" />
                                  <Select
                                    onValueChange={field.onChange}
                                    value={field.value}
                                  >
                                    <FormControl>
                                      <SelectTrigger className="w-full">
                                        <SelectValue />
                                      </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                      {QUERY_STRATEGIES.map((item) => (
                                        <SelectItem key={item} value={item}>
                                          {item}
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            <SwitchField
                              control={form.control}
                              label={t("form.disableCache", "Disable Cache")}
                              name="disable_cache"
                            />
                            <SwitchField
                              control={form.control}
                              label={t("form.serveStale", "Serve Stale")}
                              name="serve_stale"
                            />
                            <FormField
                              control={form.control}
                              name="serve_expired_ttl"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>
                                    <XrayFieldLabel fieldKey="serve_expired_ttl">
                                      {t(
                                        "form.serveExpiredTtl",
                                        "Serve Expired TTL"
                                      )}
                                    </XrayFieldLabel>
                                  </FormLabel>
                                  <XrayFieldDescription fieldKey="serve_expired_ttl" />
                                  <FormControl>
                                    <EnhancedInput
                                      inputMode="numeric"
                                      min={0}
                                      onValueChange={field.onChange}
                                      preserveZero
                                      step={1}
                                      type="number"
                                      value={field.value}
                                    />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                            <SwitchField
                              control={form.control}
                              label={t(
                                "form.disableFallback",
                                "Disable Fallback"
                              )}
                              name="disable_fallback"
                            />
                            <SwitchField
                              control={form.control}
                              label={t(
                                "form.disableFallbackIfMatch",
                                "Disable Fallback If Match"
                              )}
                              name="disable_fallback_if_match"
                            />
                            <SwitchField
                              control={form.control}
                              label={t(
                                "form.enableParallelQuery",
                                "Enable Parallel Query"
                              )}
                              name="enable_parallel_query"
                            />
                            <SwitchField
                              control={form.control}
                              label={t(
                                "form.useSystemHosts",
                                "Use System Hosts"
                              )}
                              name="use_system_hosts"
                            />
                          </div>
                          <JsonObjectField
                            addLabel="添加 Hosts 映射"
                            control={form.control}
                            description="Hosts 键值映射。值可以是文本，或选择 JSON 后填写数组。"
                            label={t("form.hostsJson", "Hosts JSON")}
                            name="hosts_json"
                          />
                        </>
                      )}
                    </TabsContent>

                    <TabsContent
                      className="xray-form-tab space-y-3 pt-4"
                      value="advanced"
                    >
                      <div className="flex flex-col gap-3 rounded-md border bg-muted/35 p-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="flex min-w-0 items-start gap-2 text-muted-foreground text-xs leading-5">
                          <Info
                            aria-hidden="true"
                            className="mt-0.5 size-4 shrink-0 text-primary"
                          />
                          <span>
                            {advancedTouched
                              ? t(
                                  "form.advancedActiveDescription",
                                  "Advanced JSON is the active configuration source. Hidden form fields will not block saving."
                                )
                              : t(
                                  "form.advancedInactiveDescription",
                                  "The structured form remains active until you edit this JSON. Sync again to discard manual JSON changes."
                                )}
                          </span>
                        </div>
                        <Button
                          className="shrink-0"
                          onClick={syncAdvancedJson}
                          size="sm"
                          type="button"
                          variant="outline"
                        >
                          {t("form.syncJson", "Sync from form")}
                        </Button>
                      </div>
                      <FormField
                        control={form.control}
                        name="advanced_json"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("form.advancedJson", "Advanced JSON")}
                            </FormLabel>
                            <FormControl>
                              <JsonTextarea
                                onChange={(value) => {
                                  setAdvancedTouched(true);
                                  field.onChange(value);
                                }}
                                value={field.value}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </TabsContent>
                    <TabsContent
                      className="xray-form-tab space-y-4 pt-4"
                      value="template"
                    >
                      <FormField
                        control={form.control}
                        name="config_template"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("form.configTemplate", "Config Template")}
                            </FormLabel>
                            <FormDescription>
                              {t(
                                "form.configTemplateDesc",
                                "Optional Go template for the final Xray JSON. Context supports Server/server, Vars/vars and Ref/ref."
                              )}
                            </FormDescription>
                            <FormControl>
                              <Textarea
                                className="min-h-52 font-mono text-xs"
                                onChange={field.onChange}
                                placeholder='{"tag":"{{ .Vars.tag }}","port":{{ .Vars.port }}}'
                                value={field.value || ""}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                        <VariableSchemaField
                          control={form.control}
                          name="variables_schema_json"
                        />
                        <JsonObjectField
                          addLabel="添加默认变量"
                          control={form.control}
                          description="这些值会作为绑定服务器时的默认变量。"
                          label={t(
                            "form.defaultVariables",
                            "Default Variables"
                          )}
                          name="default_variables_json"
                        />
                      </div>
                    </TabsContent>
                    <TabsContent
                      className="xray-form-tab space-y-4 pt-4"
                      value="subscription"
                    >
                      <JsonObjectField
                        addLabel="添加订阅字段"
                        control={form.control}
                        description={t(
                          "form.subscriptionMetaDesc",
                          "Inbound 模板可把渲染变量映射到订阅 Proxy 字段，例如 type、port、security、transport、sni。"
                        )}
                        label={t("form.subscriptionMeta", "Subscription Meta")}
                        name="subscription_meta_json"
                      />
                    </TabsContent>
                  </Tabs>
                </AdminForm>
              </Form>
            </ScrollArea>
          </div>

          <SheetFooter className="xray-template-footer flex-row justify-between gap-2">
            <span className="hidden text-muted-foreground text-xs sm:block">
              {form.formState.isDirty
                ? t("form.unsavedChanges", "Unsaved changes")
                : t("form.noChanges", "No unsaved changes")}
            </span>
            <div className="flex gap-2">
              <Button
                onClick={() => handleOpenChange(false)}
                type="button"
                variant="outline"
              >
                {t("form.cancel", "Cancel")}
              </Button>
              <Button
                disabled={loading}
                form="xray-template-form"
                type="submit"
              >
                {loading ? (
                  <Icon className="mr-2 animate-spin" icon="mdi:loading" />
                ) : null}
                {t("form.save", "Save template")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <DiscardChangesDialog
        onDiscard={discardChanges}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
