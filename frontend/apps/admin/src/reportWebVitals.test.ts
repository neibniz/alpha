import { describe, expect, it } from "vitest";
import { resolveWebVitalsEndpoint, webVitalsPayload } from "./reportWebVitals";

describe("Admin Web Vitals", () => {
  it("only resolves same-origin paths without query parameters", () => {
    const origin = "https://admin.example.test";
    expect(resolveWebVitalsEndpoint("/metrics/web-vitals", origin)).toBe(
      `${origin}/metrics/web-vitals`
    );
    for (const path of [
      "//example.test/report",
      "/\\example.test/report",
      "/metrics?token=secret",
      "/metrics#hash",
      "https://example.test/report",
    ]) {
      expect(resolveWebVitalsEndpoint(path, origin)).toBeUndefined();
    }
  });

  it("omits the performance entry details and measurement identifier", () => {
    expect(
      webVitalsPayload({
        delta: 20,
        entries: [],
        id: "test-page",
        name: "LCP",
        navigationType: "navigate",
        rating: "good",
        value: 20,
      })
    ).toEqual({ name: "LCP", rating: "good", value: 20 });
  });
});
