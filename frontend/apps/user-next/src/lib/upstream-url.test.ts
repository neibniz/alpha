import { describe, expect, test } from "bun:test";
import { buildUpstreamPath } from "./upstream-url";

describe("buildUpstreamPath", () => {
  test("targets an unprefixed Go API by default", () => {
    expect(buildUpstreamPath("v1/auth/login")).toBe("/v1/auth/login");
  });

  test("normalizes an explicitly configured prefix", () => {
    expect(buildUpstreamPath("/v1/public/order/list", "/api/")).toBe(
      "/api/v1/public/order/list"
    );
  });
});
