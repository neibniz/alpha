import { getCookie, removeCookie, setCookie } from "@workspace/ui/lib/cookies";
import { getRuntimeConfig } from "@workspace/ui/lib/runtime-config";

const LEGACY_AUTH_COOKIE_NAME = "Authorization";

let authCookieName = LEGACY_AUTH_COOKIE_NAME;

function normalizeCookieName(name?: string) {
  const trimmed = name?.trim();
  return trimmed || LEGACY_AUTH_COOKIE_NAME;
}

export function configureAuthCookieName(name: string) {
  authCookieName = normalizeCookieName(name);
}

export function getAuthCookieName() {
  return normalizeCookieName(
    getRuntimeConfig().AUTH_COOKIE_NAME || authCookieName
  );
}

export function getAuthorizationToken() {
  return getCookie(getAuthCookieName());
}

export function setAuthorizationToken(token: string) {
  setCookie(getAuthCookieName(), token);
}

export function removeAuthorizationToken() {
  removeCookie(getAuthCookieName());
}

export function clearLegacyAuthorizationToken() {
  if (getAuthCookieName() !== LEGACY_AUTH_COOKIE_NAME) {
    removeCookie(LEGACY_AUTH_COOKIE_NAME);
  }
}
