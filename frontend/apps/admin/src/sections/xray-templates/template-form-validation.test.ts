import { describe, expect, it } from "vitest";
import {
  isFiveFieldCron,
  isRealityShortId,
  isRealityShortIdList,
  isRealityTarget,
  isValidPort,
  isVlessFlowAllowed,
  isXrayVersion,
  parseOptionalInteger,
} from "./template-form-validation";

describe("Xray template form validation", () => {
  it("keeps blank optional integers unset", () => {
    expect(parseOptionalInteger("")).toBeUndefined();
    expect(parseOptionalInteger("   ")).toBeUndefined();
    expect(parseOptionalInteger("443")).toBe(443);
    expect(parseOptionalInteger("4.5")).toBe("4.5");
  });

  it("validates Xray port bounds", () => {
    expect(isValidPort(1)).toBe(true);
    expect(isValidPort("65535")).toBe(true);
    expect(isValidPort(0)).toBe(false);
    expect(isValidPort(65_536)).toBe(false);
  });

  it("accepts five-field cron expressions and template expressions", () => {
    expect(isFiveFieldCron("0 4 * * *")).toBe(true);
    expect(isFiveFieldCron("{{ .Vars.cron }}")).toBe(true);
    expect(isFiveFieldCron("0 0 4 * * *")).toBe(false);
  });

  it("validates REALITY targets", () => {
    expect(isRealityTarget("example.com:443")).toBe(true);
    expect(isRealityTarget("[2001:db8::1]:443")).toBe(true);
    expect(isRealityTarget("{{ .Vars.target }}")).toBe(true);
    expect(isRealityTarget("https://example.com")).toBe(false);
    expect(isRealityTarget("example.com")).toBe(false);
  });

  it("validates REALITY short IDs", () => {
    expect(isRealityShortId("a1b2c3d4")).toBe(true);
    expect(isRealityShortId("")).toBe(true);
    expect(isRealityShortId("abc")).toBe(false);
    expect(isRealityShortIdList("a1b2, c3d4\n0011")).toBe(true);
    expect(isRealityShortIdList("a1b2, xyz1")).toBe(false);
  });

  it("validates Xray client version fields", () => {
    expect(isXrayVersion("25.8.3")).toBe(true);
    expect(isXrayVersion("{{ .Vars.min_version }}")).toBe(true);
    expect(isXrayVersion("")).toBe(true);
    expect(isXrayVersion("25.8")).toBe(false);
    expect(isXrayVersion("v25.8.3")).toBe(false);
  });

  it("only allows VLESS flow on RAW/TCP transports", () => {
    expect(isVlessFlowAllowed("raw", "xtls-rprx-vision")).toBe(true);
    expect(isVlessFlowAllowed("tcp", "xtls-rprx-vision")).toBe(true);
    expect(isVlessFlowAllowed("xhttp", "")).toBe(true);
    expect(isVlessFlowAllowed("grpc", "none")).toBe(true);
    expect(isVlessFlowAllowed("xhttp", "xtls-rprx-vision")).toBe(false);
  });
});
