import { describe, expect, it } from "vitest";
import {
  buildMembershipPlanPayload,
  DEFAULT_MEMBERSHIP_FORM,
  membershipPlanFormKey,
  membershipPlanToForm,
  validateMembershipPlan,
} from "./membership-plan-data";

describe("membership plan data", () => {
  it("normalizes API plans without sharing policy state", () => {
    const form = membershipPlanToForm({
      id: 7,
      name: "Annual",
      description: "Plan",
      unit_price: 1200,
      duration_unit: "Year",
      duration_value: 1,
      enabled: true,
      policy: {
        monthly_consumption_limit_enabled: false,
        monthly_consumption_limit: 0,
        monthly_order_limit_enabled: false,
        monthly_order_limit: 0,
        subscription_expiry_limit_enabled: false,
        subscription_expiry_grace_days: 0,
      },
    });
    form.policy.monthly_order_limit = 4;
    expect(DEFAULT_MEMBERSHIP_FORM.policy.monthly_order_limit).toBe(0);
  });

  it("requires positive enabled policy limits", () => {
    const form = membershipPlanToForm();
    form.id = 1;
    form.name = "Annual";
    form.policy.monthly_consumption_limit_enabled = true;
    form.policy.monthly_order_limit_enabled = true;
    const errors = validateMembershipPlan(form);
    expect(errors.monthly_consumption_limit).toBeTruthy();
    expect(errors.monthly_order_limit).toBeTruthy();
  });

  it("rejects invalid integer ranges", () => {
    const form = membershipPlanToForm();
    form.name = "Annual";
    form.duration_value = 1.5;
    form.policy.subscription_expiry_grace_days = -1;
    const errors = validateMembershipPlan(form);
    expect(errors.duration_value).toBeTruthy();
    expect(errors.subscription_expiry_grace_days).toBeTruthy();
  });

  it("trims the payload and generates a stable dirty key", () => {
    const form = membershipPlanToForm();
    form.name = " Annual ";
    form.description = " Plan ";
    expect(buildMembershipPlanPayload(form).name).toBe("Annual");
    expect(membershipPlanFormKey(form)).toBe(
      membershipPlanFormKey({
        ...form,
        name: "Annual",
        description: "Plan",
      })
    );
  });
});
