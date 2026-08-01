import { describe, expect, test } from "bun:test";
import { type ApiRequester, getSiteConfigWith } from "@/lib/client-api";

describe("site config mapping", () => {
  test("preserves public site branding fields", async () => {
    const request: ApiRequester = async <T>() =>
      ({
        currency: { currency_symbol: "$" },
        oauth_methods: ["github"],
        site: {
          keywords: "science, network",
          site_desc: "A focused subscription service.",
          site_logo: "https://assets.example.com/logo.svg",
          site_name: "Science",
        },
        subscribe: {
          pan_domain: true,
          subscribe_domain: "subscribe.example.com",
          subscribe_path: "/s",
        },
      }) as T;

    const config = await getSiteConfigWith(request);

    expect(config).toEqual({
      currencySymbol: "$",
      keywords: "science, network",
      oauthMethods: ["github"],
      panDomain: true,
      siteDescription: "A focused subscription service.",
      siteLogo: "https://assets.example.com/logo.svg",
      siteName: "Science",
      subscribeDomain: "subscribe.example.com",
      subscribePath: "/s",
    });
  });

  test("uses safe defaults for absent branding", async () => {
    const request: ApiRequester = async <T>() => ({}) as T;

    const config = await getSiteConfigWith(request);

    expect(config.siteName).toBe("Alpha");
    expect(config.siteDescription).toBe("订阅与流量管理工作台");
    expect(config.siteLogo).toBe("");
    expect(config.keywords).toBe("");
  });
});
