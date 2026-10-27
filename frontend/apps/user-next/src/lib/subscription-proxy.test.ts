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

  test("uses filesystem routes for the standard subscription paths", () => {
    expect(buildSubscriptionProxyRewrites("/api/subscribe/")).toEqual([]);
    expect(buildSubscriptionProxyRewrites("/v1/subscribe/config")).toEqual([]);
  });

  test("builds one exact rewrite for a custom path without allowing a loop", () => {
    expect(buildSubscriptionProxyRewrites("/s/custom/")).toEqual([
      {
        source: "/s/custom",
        destination: "/api/subscription-proxy",
      },
    ]);
    expect(buildSubscriptionProxyRewrites("/api/subscription-proxy")).toEqual(
      []
    );
  });
});
