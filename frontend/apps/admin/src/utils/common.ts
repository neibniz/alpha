import {
  clearLegacyAuthorizationToken,
  getAuthorizationToken,
  removeAuthorizationToken,
  setAuthorizationToken,
} from "@workspace/ui/lib/auth-token";
import { getApiBaseURL, getApiPrefix } from "@workspace/ui/lib/runtime-config";
import { isBrowser } from "@workspace/ui/utils/index";

export function getPlatform(): string {
  if (typeof window === "undefined") return "unknown";

  const userAgent = navigator.userAgent.toLowerCase();

  if (userAgent.includes("win")) return "windows";
  if (userAgent.includes("mac")) return "macos";
  if (userAgent.includes("linux")) return "linux";
  if (userAgent.includes("android")) return "android";
  if (userAgent.includes("iphone") || userAgent.includes("ipad")) return "ios";

  return "unknown";
}

export { differenceInDays, formatDate } from "@workspace/ui/utils/formatting";

export function setAuthorization(token: string): void {
  setAuthorizationToken(token);
  clearLegacyAuthorizationToken();
}

export function getRedirectUrl(): string {
  if (typeof window === "undefined") return "/dashboard";
  const stored = sessionStorage.getItem("redirect-url");
  sessionStorage.removeItem("redirect-url");
  const params = new URLSearchParams(window.location.search);
  const redirect = params.get("redirect") || stored;
  return isInternalRedirect(redirect) ? redirect : "/dashboard";
}

export function setRedirectUrl(value?: string) {
  if (isInternalRedirect(value)) {
    sessionStorage.setItem("redirect-url", value);
  }
}

export function isInternalRedirect(value?: string | null): value is string {
  return Boolean(value?.startsWith("/") && !value.startsWith("//"));
}

export function Logout() {
  if (!isBrowser()) return;
  const token = getAuthorizationToken();
  if (token) {
    const base = getApiBaseURL() || window.location.origin;
    const url = new URL(`${getApiPrefix()}/v1/auth/logout`, base);
    fetch(url.toString(), {
      headers: { Authorization: token },
      keepalive: true,
      method: "POST",
    }).catch((error) => {
      console.debug("Failed to report logout:", error?.message || error);
    });
  }
  removeAuthorizationToken();
  clearLegacyAuthorizationToken();

  const pathname = location.pathname;
  const hash = location.hash.slice(1);

  if (!["", "/"].includes(pathname)) {
    setRedirectUrl(pathname);
    location.href = "/";
    return;
  }

  if (hash && !["", "/"].includes(hash)) {
    setRedirectUrl(hash);
    location.href = "/";
  }
}
