import { describe, expect, it } from "vitest";
import {
  isValidDomainLine,
  isValidSubscribePath,
  isValidUserAgentLine,
  lineCount,
  normalizeLines,
  normalizeSubscribeConfig,
  normalizeSubscribePath,
  subscriptionEndpointPreview,
  subscriptionConfigTabForFields,
} from "./subscription-config-data";

describe("subscription configuration data", () => {
  it("normalizes line lists without changing entry order", () => {
    expect(normalizeLines(" ClashX \r\nsing-box\nClashX\n\n")).toBe(
      "ClashX\nsing-box"
    );
    expect(lineCount("ClashX\nClashX\nsing-box")).toBe(2);
  });

  it("normalizes one leading slash and accepts static route segments", () => {
    expect(normalizeSubscribePath(" v1/subscribe/config ")).toBe(
      "/v1/subscribe/config"
    );
    expect(normalizeSubscribePath("///alpha-sub")).toBe("/alpha-sub");
    expect(isValidSubscribePath("/v1/subscribe/config")).toBe(true);
    expect(isValidSubscribePath("/alpha_sub-v2")).toBe(true);
  });

  it("rejects paths that Gin would interpret dynamically or ambiguously", () => {
    for (const value of [
      "/",
      "/sub/",
      "/sub//config",
      "/sub/:token",
      "/sub/*path",
      "/sub?token=1",
      "/sub#fragment",
      "/sub config",
    ]) {
      expect(isValidSubscribePath(value), value).toBe(false);
    }
  });

  it("accepts concrete hostnames and valid IPv4 addresses", () => {
    expect(isValidDomainLine("subscribe.example.com")).toBe(true);
    expect(isValidDomainLine("localhost")).toBe(true);
    expect(isValidDomainLine("192.0.2.192")).toBe(true);
  });

  it("rejects wildcards, protocols, ports, paths, and malformed hostnames", () => {
    for (const value of [
      "*.edge.example.com",
      "https://subscribe.example.com",
      "subscribe.example.com:8443",
      "subscribe.example.com/path",
      "-edge.example.com",
      "edge..example.com",
      "999.168.1.1",
    ]) {
      expect(isValidDomainLine(value), value).toBe(false);
    }
  });

  it("accepts printable User-Agent matchers and rejects control characters", () => {
    expect(isValidUserAgentLine("Clash Verge Rev")).toBe(true);
    expect(isValidUserAgentLine("sing-box/1.12")).toBe(true);
    expect(isValidUserAgentLine("客".repeat(255))).toBe(true);
    expect(isValidUserAgentLine("")).toBe(false);
    expect(isValidUserAgentLine(`Clash${String.fromCharCode(0)}Meta`)).toBe(
      false
    );
    expect(isValidUserAgentLine("x".repeat(256))).toBe(false);
  });

  it("normalizes every list before sending the API payload", () => {
    expect(
      normalizeSubscribeConfig({
        single_model: false,
        pan_domain: true,
        subscribe_path: "sub/config",
        subscribe_domain: "sub.example.com\nsub.example.com",
        user_agent_limit: true,
        user_agent_list: "Clash\n Clash ",
      })
    ).toEqual({
      single_model: false,
      pan_domain: true,
      subscribe_path: "/sub/config",
      subscribe_domain: "sub.example.com",
      user_agent_limit: true,
      user_agent_list: "Clash",
    });
  });

  it("routes validation errors to the matching settings tab", () => {
    expect(subscriptionConfigTabForFields(["single_model"])).toBe("policy");
    expect(subscriptionConfigTabForFields(["subscribe_path"])).toBe("endpoint");
    expect(subscriptionConfigTabForFields(["user_agent_list"])).toBe("access");
  });

  it("builds a readable endpoint preview for standard and wildcard modes", () => {
    const base = {
      single_model: false,
      pan_domain: false,
      subscribe_path: "/sub/config",
      subscribe_domain: "sub.example.com\nbackup.example.com",
      user_agent_limit: false,
      user_agent_list: "",
    };
    expect(subscriptionEndpointPreview(base)).toBe(
      "https://sub.example.com/sub/config?token=••••••••"
    );
    expect(subscriptionEndpointPreview({ ...base, pan_domain: true })).toBe(
      "https://{token}.{client}.sub.example.com/sub/config?token=••••••••"
    );
  });
});
