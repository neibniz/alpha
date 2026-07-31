export type RuntimeConfig = {
  API_BASE_URL?: string;
  API_PREFIX?: string;
  AUTH_COOKIE_NAME?: string;
};

export function getRuntimeConfig(): RuntimeConfig {
  if (typeof window === "undefined") return {};
  return window.__APP_CONFIG__ || {};
}

export function getApiBaseURL() {
  return getRuntimeConfig().API_BASE_URL;
}

export function getApiPrefix() {
  const config = getRuntimeConfig();
  if (Object.hasOwn(config, "API_PREFIX")) {
    return config.API_PREFIX || "";
  }
  return import.meta.env.DEV ? "/api" : "";
}
