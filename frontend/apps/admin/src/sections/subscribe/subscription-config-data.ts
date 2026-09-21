export const SUBSCRIBE_CONFIG_QUERY_KEY = ["getSubscribeConfig"] as const;
export type SubscribeConfigTab = "policy" | "endpoint" | "access";

export function subscriptionConfigTabForFields(
  fields: string[]
): SubscribeConfigTab {
  if (fields.some((field) => ["single_model", "pan_domain"].includes(field))) {
    return "policy";
  }
  if (
    fields.some((field) =>
      ["subscribe_path", "subscribe_domain"].includes(field)
    )
  ) {
    return "endpoint";
  }
  return "access";
}

export const DEFAULT_SUBSCRIBE_CONFIG: API.SubscribeConfig = {
  single_model: false,
  pan_domain: false,
  subscribe_path: "",
  subscribe_domain: "",
  user_agent_limit: false,
  user_agent_list: "",
};

const subscribeConfigResponseSchema = z.object({
  single_model: z.boolean(),
  pan_domain: z.boolean(),
  subscribe_path: z.string(),
  subscribe_domain: z.string(),
  user_agent_limit: z.boolean(),
  user_agent_list: z.string(),
});

export function parseSubscribeConfigResponse(
  value: unknown
): API.SubscribeConfig {
  return subscribeConfigResponseSchema.parse(value);
}

const SAFE_PATH_PATTERN = /^\/[A-Za-z0-9._~-]+(?:\/[A-Za-z0-9._~-]+)*$/;
const DOMAIN_LABEL_PATTERN = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/;
const CONTROL_CHARACTER_PATTERN = /\p{Cc}/u;

export function normalizeLines(value: string) {
  return Array.from(
    new Set(
      value
        .split("\n")
        .map((item) => item.trim())
        .filter(Boolean)
    )
  ).join("\n");
}

export function lineCount(value?: string) {
  if (!value) return 0;
  return normalizeLines(value).split("\n").filter(Boolean).length;
}

export function normalizeSubscribePath(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return "";
  return `/${trimmed.replace(/^\/+/, "")}`;
}

export function isValidSubscribePath(value: string) {
  if (!value) return true;
  return value.length <= 160 && SAFE_PATH_PATTERN.test(value);
}

function isValidIPv4(value: string) {
  const octets = value.split(".");
  return (
    octets.length === 4 &&
    octets.every((octet) => {
      if (!/^\d{1,3}$/.test(octet)) return false;
      const number = Number(octet);
      return number >= 0 && number <= 255;
    })
  );
}

export function isValidDomainLine(value: string) {
  const candidate = value.trim();
  if (!candidate || candidate.length > 255) return false;
  if (/[:/@?#\\\s]/.test(candidate)) return false;

  if (candidate.includes("*")) return false;
  if (candidate.length > 253) return false;
  if (/^\d+(?:\.\d+){3}$/.test(candidate)) return isValidIPv4(candidate);

  return candidate
    .split(".")
    .every((label) => DOMAIN_LABEL_PATTERN.test(label));
}

export function isValidUserAgentLine(value: string) {
  const candidate = value.trim();
  const characterCount = Array.from(candidate).length;
  return (
    characterCount > 0 &&
    characterCount <= 255 &&
    !CONTROL_CHARACTER_PATTERN.test(candidate)
  );
}

export function normalizeSubscribeConfig(
  values: API.SubscribeConfig
): API.SubscribeConfig {
  return {
    ...values,
    subscribe_path: normalizeSubscribePath(values.subscribe_path),
    subscribe_domain: normalizeLines(values.subscribe_domain),
    user_agent_list: normalizeLines(values.user_agent_list),
  };
}

export function subscriptionEndpointPreview(values: API.SubscribeConfig) {
  const [configuredDomain] = normalizeLines(values.subscribe_domain).split(
    "\n"
  );
  const domain = configuredDomain || "your-domain.example";
  const host = values.pan_domain ? `{token}.{client}.${domain}` : domain;
  const path =
    normalizeSubscribePath(values.subscribe_path) || "/v1/subscribe/config";
  return `https://${host}${path}?token=••••••••`;
}
import { z } from "zod";
