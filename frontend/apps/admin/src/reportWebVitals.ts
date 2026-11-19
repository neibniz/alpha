import type { Metric } from "web-vitals";

export function resolveWebVitalsEndpoint(path: string, origin: string) {
  if (!path.startsWith("/") || path.startsWith("//")) return;
  try {
    const url = new URL(path, origin);
    if (url.origin === origin && !url.search && !url.hash) return url.href;
  } catch {
    return;
  }
}

export function webVitalsPayload({ name, rating, value }: Metric) {
  return { name, rating, value };
}

export default function reportWebVitals(path?: string) {
  if (!path?.trim()) return;
  const endpoint = resolveWebVitalsEndpoint(path.trim(), location.origin);
  if (!endpoint) return;

  const report = (metric: Metric) => {
    fetch(endpoint, {
      body: JSON.stringify(webVitalsPayload(metric)),
      cache: "no-store",
      credentials: "omit",
      headers: { "Content-Type": "application/json" },
      keepalive: true,
      method: "POST",
      redirect: "error",
      referrerPolicy: "no-referrer",
    }).catch(() => undefined);
  };

  import("web-vitals")
    .then(({ onCLS, onINP, onLCP }) => {
      onCLS(report);
      onINP(report);
      onLCP(report);
    })
    .catch(() => undefined);
}
