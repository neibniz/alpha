const BUILT_IN_SUBSCRIPTION_PATHS = new Set([
  "/api/subscribe",
  "/v1/subscribe/config",
]);
const INTERNAL_SUBSCRIPTION_PROXY_PATH = "/api/subscription-proxy";

export function normalizeSubscriptionProxyPath(value: string | undefined) {
  const trimmed = value?.trim() || "/v1/subscribe/config";
  const path = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return path.length > 1 ? path.replace(/\/+$/g, "") : path;
}

export function buildSubscriptionProxyRewrites(
  configuredPath: string | undefined
) {
  const source = normalizeSubscriptionProxyPath(configuredPath);
  if (
    source === INTERNAL_SUBSCRIPTION_PROXY_PATH ||
    BUILT_IN_SUBSCRIPTION_PATHS.has(source)
  ) {
    return [];
  }
  return [{ source, destination: INTERNAL_SUBSCRIPTION_PROXY_PATH }];
}
