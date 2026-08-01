import { describe, expect, test } from "bun:test";
import type { SiteConfig, Subscription } from "@/lib/models";
import { buildSubscriptionUrls } from "@/lib/subscription-url";

const subscription = {
  short: "alpha7k",
  token: "private-token",
} as Subscription;

const config: SiteConfig = {
  currencySymbol: "¥",
  keywords: "",
  oauthMethods: ["github"],
  panDomain: false,
  siteDescription: "",
  siteLogo: "",
  siteName: "Alpha",
  subscribeDomain: "https://sub-a.example.com\nsub-b.example.com",
  subscribePath: "v1/subscribe/config",
};

describe("buildSubscriptionUrls", () => {
  test("preserves configured schemes and all unique domains", () => {
    expect(
      buildSubscriptionUrls(
        subscription,
        config,
        "ps_1",
        "https://user.example.com"
      )
    ).toEqual([
      "https://sub-a.example.com/v1/subscribe/config?token=private-token&preset=ps_1",
      "https://sub-b.example.com/v1/subscribe/config?token=private-token&preset=ps_1",
    ]);
  });

  test("uses the subscription short code for pan domains", () => {
    expect(
      buildSubscriptionUrls(
        subscription,
        { ...config, panDomain: true, subscribeDomain: "sub.example.com" },
        undefined,
        "https://user.example.com"
      )[0]
    ).toBe(
      "https://alpha7k.sub.example.com/v1/subscribe/config?token=private-token"
    );
  });

  test("creates a distinct subscription URL for each preset key", () => {
    const workPreset = buildSubscriptionUrls(
      subscription,
      config,
      "ps_work",
      "https://user.example.com"
    )[0];
    const mobilePreset = buildSubscriptionUrls(
      subscription,
      config,
      "ps_mobile",
      "https://user.example.com"
    )[0];

    expect(workPreset === mobilePreset).toBe(false);
    expect(new URL(workPreset || "").searchParams.get("preset")).toBe(
      "ps_work"
    );
    expect(new URL(mobilePreset || "").searchParams.get("preset")).toBe(
      "ps_mobile"
    );
  });

  test("falls back to the current origin without creating localhost subdomains", () => {
    expect(
      buildSubscriptionUrls(
        subscription,
        { ...config, panDomain: true, subscribeDomain: "" },
        undefined,
        "http://localhost:3100"
      )[0]
    ).toBe("http://localhost:3100/v1/subscribe/config?token=private-token");
  });
});
