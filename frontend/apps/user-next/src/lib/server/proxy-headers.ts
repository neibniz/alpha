const TRUSTED_PROXY_HEADERS = [
  "x-forwarded-for",
  "x-forwarded-proto",
  "x-real-ip",
] as const;

export function upstreamRequestHeaders(
  source: Headers,
  initial?: HeadersInit,
  trustProxyHeaders = false
) {
  const headers = new Headers(initial);
  const userAgent = source.get("user-agent")?.trim();
  if (userAgent) headers.set("User-Agent", userAgent);

  if (!trustProxyHeaders) return headers;

  for (const name of TRUSTED_PROXY_HEADERS) {
    const value = source.get(name)?.trim();
    if (value) headers.set(name, value);
  }
  return headers;
}
