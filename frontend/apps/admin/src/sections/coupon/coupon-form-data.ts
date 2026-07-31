import * as z from "zod";

const MAX_AMOUNT = 2_000_000_000;
const COUPON_CODE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{2,63}$/;

type CouponTranslator = (key: string, defaultValue: string) => string;

const defaultTranslator: CouponTranslator = (_key, defaultValue) =>
  defaultValue;

export const createCouponFormSchema = (
  t: CouponTranslator = defaultTranslator
) =>
  z
    .object({
      id: z.number().int().positive().optional(),
      name: z
        .string()
        .trim()
        .min(1, {
          message: t("form.validation.nameRequired", "Enter a coupon name."),
        })
        .max(255, {
          message: t(
            "form.validation.nameTooLong",
            "Coupon name cannot exceed 255 characters."
          ),
        }),
      code: z
        .string()
        .trim()
        .refine((value) => !value || COUPON_CODE_PATTERN.test(value), {
          message: t(
            "form.validation.code",
            "Coupon code must contain 3 to 64 letters, numbers, underscores, or hyphens."
          ),
        }),
      count: z.number().int().min(0).max(MAX_AMOUNT),
      type: z.union([z.literal(1), z.literal(2)]),
      discount: z.number().int().min(1).max(MAX_AMOUNT),
      start_time: z.number().int().positive(),
      expire_time: z.number().int().positive(),
      subscribe: z.array(z.number().int().positive()),
      user_limit: z.number().int().min(0).max(MAX_AMOUNT),
      used_count: z.number().int().min(0).optional(),
      enable: z.boolean().optional(),
    })
    .superRefine((value, context) => {
      if (value.type === 1 && value.discount > 100) {
        context.addIssue({
          code: "custom",
          message: t(
            "form.validation.percentage",
            "Percentage discount must be between 1% and 100%."
          ),
          path: ["discount"],
        });
      }
      if (value.expire_time <= value.start_time) {
        context.addIssue({
          code: "custom",
          message: t(
            "form.validation.expiry",
            "Expiry time must be later than the start time."
          ),
          path: ["expire_time"],
        });
      }
      if (value.count > 0 && (value.used_count || 0) > value.count) {
        context.addIssue({
          code: "custom",
          message: t(
            "form.validation.count",
            "Total usage limit cannot be lower than the used count."
          ),
          path: ["count"],
        });
      }
    });

export const couponFormSchema = createCouponFormSchema();

export type CouponFormValues = z.infer<typeof couponFormSchema>;
export type CouponFormTab = "discount" | "limits";

export function defaultCouponFormValues(now = Date.now()): CouponFormValues {
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  const expire = new Date(start);
  expire.setDate(expire.getDate() + 30);
  return {
    name: "",
    code: "",
    count: 0,
    type: 1,
    discount: 10,
    start_time: start.getTime(),
    expire_time: expire.getTime(),
    subscribe: [],
    user_limit: 0,
    used_count: 0,
    enable: false,
  };
}

export function couponToFormValues(
  coupon?: Partial<API.Coupon>
): CouponFormValues {
  return {
    ...defaultCouponFormValues(),
    ...coupon,
    name: coupon?.name || "",
    code: coupon?.code || "",
    count: coupon?.count ?? 0,
    type: coupon?.type === 2 ? 2 : 1,
    discount: coupon?.discount || (coupon?.type === 2 ? 100 : 10),
    subscribe: coupon?.subscribe || [],
    user_limit: coupon?.user_limit ?? 0,
    used_count: coupon?.used_count ?? 0,
    enable: coupon?.enable ?? false,
  };
}

export function couponErrorTab(fields: string[]): CouponFormTab {
  if (
    fields.some((field) =>
      ["start_time", "expire_time", "count", "user_limit"].includes(field)
    )
  ) {
    return "limits";
  }
  return "discount";
}

export function buildCreateCouponPayload(
  values: CouponFormValues
): API.CreateCouponRequest {
  return {
    name: values.name.trim(),
    code: values.code.trim() || undefined,
    count: values.count,
    type: values.type,
    discount: values.discount,
    start_time: values.start_time,
    expire_time: values.expire_time,
    user_limit: values.user_limit,
    subscribe: values.subscribe,
    used_count: 0,
    enable: false,
  };
}

export function buildUpdateCouponPayload(
  values: CouponFormValues
): API.UpdateCouponRequest {
  if (!values.id) throw new Error("Coupon id is missing");
  return {
    id: values.id,
    name: values.name.trim(),
    code: values.code.trim(),
    count: values.count,
    type: values.type,
    discount: values.discount,
    start_time: values.start_time,
    expire_time: values.expire_time,
    user_limit: values.user_limit,
    subscribe: values.subscribe,
    used_count: values.used_count || 0,
    enable: values.enable ?? false,
  };
}

export type CouponLifecycle =
  | "disabled"
  | "scheduled"
  | "active"
  | "expired"
  | "exhausted";

function timestampMilliseconds(value: number) {
  return value < 100_000_000_000 ? value * 1000 : value;
}

export function couponLifecycle(
  coupon: Pick<
    API.CouponListItem,
    "enable" | "start_time" | "expire_time" | "count" | "used_count"
  >,
  now = Date.now()
): CouponLifecycle {
  if (!coupon.enable) return "disabled";
  if (coupon.count > 0 && coupon.used_count >= coupon.count) return "exhausted";
  if (now < timestampMilliseconds(coupon.start_time)) return "scheduled";
  if (now >= timestampMilliseconds(coupon.expire_time)) return "expired";
  return "active";
}
