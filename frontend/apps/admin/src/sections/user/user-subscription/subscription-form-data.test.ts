import { describe, expect, test } from "vitest";
import { subscriptionFormSchema } from "./subscription-form-data";

const valid = {
  subscribe_id: 1,
  traffic: 0,
  expired_at: 0,
  upload: 0,
  download: 0,
};

describe("subscription form validation", () => {
  test("accepts zero values used for unlimited and permanent subscriptions", () => {
    expect(subscriptionFormSchema.safeParse(valid).success).toBe(true);
  });

  test.each([
    "traffic",
    "expired_at",
    "upload",
    "download",
  ] as const)("rejects negative %s", (field) => {
    expect(
      subscriptionFormSchema.safeParse({ ...valid, [field]: -1 }).success
    ).toBe(false);
  });

  test("rejects a missing or invalid product", () => {
    expect(
      subscriptionFormSchema.safeParse({ ...valid, subscribe_id: 0 }).success
    ).toBe(false);
    const { subscribe_id: _, ...missingProduct } = valid;
    expect(subscriptionFormSchema.safeParse(missingProduct).success).toBe(
      false
    );
  });

  test("rejects unsafe integers before JSON loses precision", () => {
    expect(
      subscriptionFormSchema.safeParse({
        ...valid,
        traffic: Number.MAX_SAFE_INTEGER + 1,
      }).success
    ).toBe(false);
  });
});
