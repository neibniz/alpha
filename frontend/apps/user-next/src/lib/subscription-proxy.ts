export function normalizeSubscriptionProxyPath(value: string | undefined) {
  const trimmed = value?.trim() || "/v1/subscribe/config";
  const path = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return path.length > 1 ? path.replace(/\/+$/g, "") : path;
}

export function buildSubscriptionProxyRewrites(
  configuredPath: string | undefined
) {
  const source = normalizeSubscriptionProxyPath(configuredPath);
  if (source === "/api/subscription-proxy") return [];
  return [{ source, destination: "/api/subscription-proxy" }];
}
