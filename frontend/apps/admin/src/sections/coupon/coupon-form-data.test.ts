import { describe, expect, it } from "vitest";
import {
  buildCreateCouponPayload,
  couponErrorTab,
  couponFormSchema,
  couponLifecycle,
  couponToFormValues,
  defaultCouponFormValues,
} from "./coupon-form-data";

describe("coupon form data", () => {
  it("creates a valid 30-day default period", () => {
    const form = defaultCouponFormValues(Date.UTC(2026, 6, 12, 12));
    expect(form.expire_time).toBeGreaterThan(form.start_time);
    expect(
      couponFormSchema.safeParse({ ...form, name: "Launch" }).success
    ).toBe(true);
  });

  it("validates percentage, code, dates, and usage counts", () => {
    const form = defaultCouponFormValues();
    form.name = "Summer";
    form.code = "bad code";
    form.discount = 101;
    form.expire_time = form.start_time;
    form.count = 2;
    form.used_count = 3;
    const result = couponFormSchema.safeParse(form);
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((issue) => issue.path[0]);
      expect(fields).toEqual(
        expect.arrayContaining(["code", "discount", "expire_time", "count"])
      );
    }
  });

  it("preserves edit-only state and resets create usage", () => {
    const form = couponToFormValues({
      id: 7,
      name: "SAVE",
      code: "SAVE-20",
      count: 10,
      type: 1,
      discount: 20,
      start_time: 1_700_000_000_000,
      expire_time: 1_800_000_000_000,
      user_limit: 1,
      subscribe: [2],
      used_count: 4,
      enable: true,
    });
    expect(form.used_count).toBe(4);
    expect(buildCreateCouponPayload(form).used_count).toBe(0);
    expect(buildCreateCouponPayload(form).enable).toBe(false);
  });

  it("maps validation errors to the relevant tab", () => {
    expect(couponErrorTab(["discount"])).toBe("discount");
    expect(couponErrorTab(["expire_time"])).toBe("limits");
  });

  it("derives lifecycle using seconds or milliseconds", () => {
    const now = Date.UTC(2026, 6, 12);
    const base = {
      enable: true,
      count: 10,
      used_count: 1,
      start_time: Math.floor((now - 1000) / 1000),
      expire_time: now + 1000,
    };
    expect(couponLifecycle(base, now)).toBe("active");
    expect(couponLifecycle({ ...base, enable: false }, now)).toBe("disabled");
    expect(couponLifecycle({ ...base, used_count: 10 }, now)).toBe("exhausted");
  });
});
