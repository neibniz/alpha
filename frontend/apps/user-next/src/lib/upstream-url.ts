export function buildUpstreamPath(path: string, prefix = "") {
  const normalizedPath = path.replace(/^\/+/, "");
  const normalizedPrefix = prefix.trim().replace(/^\/+|\/+$/g, "");
  return `/${[normalizedPrefix, normalizedPath].filter(Boolean).join("/")}`;
}
