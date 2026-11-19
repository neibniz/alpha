import { describe, expect, it } from "vitest";
import {
  hasAdminVerificationConfig,
  isUnauthorizedSessionError,
} from "./admin-bootstrap";

describe("administrator bootstrap", () => {
  it("fails closed until verification rules and any required site key are loaded", () => {
    const config = {
      auth: { email: {}, register: {} },
      invite: {},
      verify: {
        enable_login_verify: false,
        enable_register_verify: false,
        enable_reset_password_verify: false,
        turnstile_site_key: "",
      },
    } as API.GetGlobalConfigResponse;
    expect(hasAdminVerificationConfig(undefined)).toBe(false);
    expect(
      hasAdminVerificationConfig({
        ...config,
        verify: {} as typeof config.verify,
      })
    ).toBe(false);
    expect(hasAdminVerificationConfig(config)).toBe(true);
    config.verify.enable_reset_password_verify = true;
    expect(hasAdminVerificationConfig(config)).toBe(false);
    config.verify.turnstile_site_key = "test-site-key";
    expect(hasAdminVerificationConfig(config)).toBe(true);
  });

  it("recognizes direct and nested unauthorized responses", () => {
    expect(isUnauthorizedSessionError({ status: 401 })).toBe(true);
    expect(isUnauthorizedSessionError({ data: { code: 40_004 } })).toBe(true);
    expect(
      isUnauthorizedSessionError({
        response: { data: { code: 40_003 }, status: 200 },
      })
    ).toBe(true);
  });

  it("does not turn transient failures into logout events", () => {
    expect(
      isUnauthorizedSessionError({ status: 200, data: { code: 401 } })
    ).toBe(false);
    expect(
      isUnauthorizedSessionError({
        response: { status: 200, data: { code: 401 } },
      })
    ).toBe(false);
    expect(isUnauthorizedSessionError({ status: 500 })).toBe(false);
    expect(
      isUnauthorizedSessionError({
        response: { data: { code: 500 }, status: 503 },
      })
    ).toBe(false);
    expect(isUnauthorizedSessionError(new Error("Network unavailable"))).toBe(
      false
    );
  });
});
