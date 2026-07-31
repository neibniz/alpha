import { describe, expect, test } from "bun:test";
import {
  buildSubscriptionProxyRewrites,
  normalizeSubscriptionProxyPath,
} from "./subscription-proxy";

describe("subscription proxy routing", () => {
  test("normalizes the administrator-configured path", () => {
    expect(normalizeSubscriptionProxyPath("api/subscribe/")).toBe(
      "/api/subscribe"
    );
  });

  test("builds one exact rewrite without allowing a proxy loop", () => {
    expect(buildSubscriptionProxyRewrites("/api/subscribe/")).toEqual([
      {
        source: "/api/subscribe",
        destination: "/api/subscription-proxy",
      },
    ]);
    expect(buildSubscriptionProxyRewrites("/api/subscription-proxy")).toEqual(
      []
    );
  });
});
