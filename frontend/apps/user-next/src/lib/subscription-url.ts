import type { SiteConfig, Subscription } from "@/lib/models";

const SCHEME_PATTERN = /^[a-z][a-z\d+.-]*:\/\//i;

function normalizeEndpoint(value: string, fallbackProtocol = "https:") {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  try {
    const parsed = new URL(
      SCHEME_PATTERN.test(trimmed) ? trimmed : `${fallbackProtocol}//${trimmed}`
    );
    if (!(parsed.protocol === "http:" || parsed.protocol === "https:")) {
      return undefined;
    }
    return { host: parsed.host, protocol: parsed.protocol };
  } catch {
    return undefined;
  }
}

function uniqueEndpoints(
  configured: string,
  origin: string
): Array<{ host: string; protocol: string }> {
  const fallback = normalizeEndpoint(origin, "https:");
  const endpoints = configured
    .split(/\r?\n/)
    .map((value) => normalizeEndpoint(value, fallback?.protocol || "https:"))
    .filter((value): value is { host: string; protocol: string } =>
      Boolean(value)
    );
  if (endpoints.length === 0 && fallback) endpoints.push(fallback);

  const seen = new Set<string>();
  return endpoints.filter((endpoint) => {
    const key = `${endpoint.protocol}//${endpoint.host}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function normalizePath(value: string) {
  const path = value.trim() || "/s";
  return path.startsWith("/") ? path : `/${path}`;
}

export function buildSubscriptionUrls(
  subscription: Subscription,
  config: SiteConfig | undefined,
  preset?: string,
  origin = typeof window === "undefined" ? "" : window.location.origin
) {
  const endpoints = uniqueEndpoints(config?.subscribeDomain || "", origin);
  const params = new URLSearchParams({ token: subscription.token });
  if (preset) params.set("preset", preset);
  const path = normalizePath(config?.subscribePath || "");

  return endpoints.map((endpoint) => {
    const canUsePanDomain =
      Boolean(config?.panDomain && subscription.short) &&
      !endpoint.host.includes(":") &&
      endpoint.host !== "localhost" &&
      !/^\d+(?:\.\d+){3}$/.test(endpoint.host);
    const host = canUsePanDomain
      ? `${subscription.short}.${endpoint.host}`
      : endpoint.host;
    return `${endpoint.protocol}//${host}${path}?${params.toString()}`;
  });
}

export function buildSubscriptionUrl(
  subscription: Subscription,
  config: SiteConfig | undefined,
  preset?: string,
  origin?: string
) {
  return buildSubscriptionUrls(subscription, config, preset, origin)[0] || "";
}
