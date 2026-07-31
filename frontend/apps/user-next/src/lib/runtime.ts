export function isClientDemoMode() {
  if (process.env.NODE_ENV === "production") return false;
  return (
    process.env.NEXT_PUBLIC_ALPHA_DEMO === "true" ||
    (process.env.NODE_ENV === "development" &&
      process.env.NEXT_PUBLIC_ALPHA_DEMO !== "false")
  );
}

export function isServerDemoMode() {
  if (process.env.NODE_ENV === "production") return false;
  return (
    process.env.NEXT_PUBLIC_ALPHA_DEMO === "true" ||
    (process.env.NODE_ENV === "development" &&
      process.env.NEXT_PUBLIC_ALPHA_DEMO !== "false")
  );
}

export function authCookieName() {
  return (
    process.env.ALPHA_AUTH_COOKIE_NAME?.trim() || "alpha_user_authorization"
  );
}

export function authCookieMaxAge() {
  const configured = Number(process.env.ALPHA_AUTH_COOKIE_MAX_AGE || 3600);
  if (!Number.isFinite(configured)) return 3600;
  return Math.min(60 * 60 * 24 * 30, Math.max(300, Math.floor(configured)));
}

export function authCookieSecure() {
  const configured = process.env.ALPHA_AUTH_COOKIE_SECURE?.trim().toLowerCase();
  if (configured === "true") return true;
  if (configured === "false") return false;
  return process.env.NODE_ENV === "production";
}
