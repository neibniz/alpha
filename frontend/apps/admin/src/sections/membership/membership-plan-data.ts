export type MembershipPlanForm = API.UpdateMembershipPlanRequest;

export type MembershipPlanErrors = Partial<
  Record<
    | "name"
    | "description"
    | "unit_price"
    | "duration_value"
    | "monthly_consumption_limit"
    | "monthly_order_limit"
    | "subscription_expiry_grace_days",
    MembershipPlanErrorCode
  >
>;

export type MembershipPlanErrorCode =
  | "validation.description"
  | "validation.durationValue"
  | "validation.monthlyConsumption"
  | "validation.monthlyOrders"
  | "validation.name"
  | "validation.price"
  | "validation.subscriptionExpiryGraceDays";

const MAX_AMOUNT = 2_000_000_000;

export const DEFAULT_MEMBERSHIP_FORM: MembershipPlanForm = {
  id: 0,
  name: "",
  description: "",
  unit_price: 0,
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
};

export function membershipPlanToForm(
  plan?: API.MembershipPlan | API.UpdateMembershipPlanRequest
): MembershipPlanForm {
  if (!plan) return structuredClone(DEFAULT_MEMBERSHIP_FORM);
  return {
    id: plan.id,
    name: plan.name || "",
    description: plan.description || "",
    unit_price: plan.unit_price,
    duration_unit: plan.duration_unit || "Year",
    duration_value: plan.duration_value || 1,
    enabled: plan.enabled,
    policy: {
      monthly_consumption_limit_enabled:
        plan.policy?.monthly_consumption_limit_enabled ?? false,
      monthly_consumption_limit: plan.policy?.monthly_consumption_limit ?? 0,
      monthly_order_limit_enabled:
        plan.policy?.monthly_order_limit_enabled ?? false,
      monthly_order_limit: plan.policy?.monthly_order_limit ?? 0,
      subscription_expiry_limit_enabled:
        plan.policy?.subscription_expiry_limit_enabled ?? false,
      subscription_expiry_grace_days:
        plan.policy?.subscription_expiry_grace_days ?? 0,
    },
  };
}

function isIntegerInRange(value: number, min: number, max: number) {
  return Number.isInteger(value) && value >= min && value <= max;
}

export function validateMembershipPlan(
  form: MembershipPlanForm
): MembershipPlanErrors {
  const errors: MembershipPlanErrors = {};
  const nameLength = Array.from(form.name.trim()).length;
  const descriptionLength = Array.from(form.description?.trim() || "").length;
  if (nameLength < 1 || nameLength > 255) {
    errors.name = "validation.name";
  }
  if (descriptionLength > 5000) {
    errors.description = "validation.description";
  }
  if (!isIntegerInRange(form.unit_price, 0, MAX_AMOUNT)) {
    errors.unit_price = "validation.price";
  }
  if (!isIntegerInRange(form.duration_value, 1, 1000)) {
    errors.duration_value = "validation.durationValue";
  }
  if (
    !isIntegerInRange(form.policy.monthly_consumption_limit, 0, MAX_AMOUNT) ||
    (form.policy.monthly_consumption_limit_enabled &&
      form.policy.monthly_consumption_limit === 0)
  ) {
    errors.monthly_consumption_limit = "validation.monthlyConsumption";
  }
  if (
    !isIntegerInRange(form.policy.monthly_order_limit, 0, 1_000_000) ||
    (form.policy.monthly_order_limit_enabled &&
      form.policy.monthly_order_limit === 0)
  ) {
    errors.monthly_order_limit = "validation.monthlyOrders";
  }
  if (
    !isIntegerInRange(form.policy.subscription_expiry_grace_days, 0, 36_500)
  ) {
    errors.subscription_expiry_grace_days =
      "validation.subscriptionExpiryGraceDays";
  }
  return errors;
}

export function buildMembershipPlanPayload(
  form: MembershipPlanForm
): MembershipPlanForm {
  return {
    ...form,
    name: form.name.trim(),
    description: form.description?.trim() || "",
    policy: { ...form.policy },
  };
}

export function membershipPlanFormKey(form: MembershipPlanForm) {
  return JSON.stringify(buildMembershipPlanPayload(form));
}
