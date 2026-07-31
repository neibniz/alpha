import { describe, expect, test } from "bun:test";
import { getMembershipCardWith } from "./client-api";

describe("membership API mapping", () => {
  test("maps feature state, plan policy and monthly usage", async () => {
    async function request<T>(path: string): Promise<T> {
      expect(path).toBe("v1/public/order/membership/card");
      return {
        enabled: true,
        expired_at: 1_800_000_000_000,
        is_member: true,
        plan: {
          description: "Account benefits",
          duration_unit: "Month",
          duration_value: 3,
          enabled: true,
          id: 7,
          name: "Quarterly member",
          policy: {
            monthly_consumption_limit: 25_000,
            monthly_consumption_limit_enabled: true,
            monthly_order_limit: 8,
            monthly_order_limit_enabled: true,
            subscription_expiry_grace_days: 15,
            subscription_expiry_limit_enabled: true,
          },
          unit_price: 2400,
        },
        usage: {
          monthly_consumption: 6800,
          monthly_order_count: 2,
        },
      } as T;
    }

    const card = await getMembershipCardWith(request);

    expect(card.enabled).toBe(true);
    expect(card.isMember).toBe(true);
    expect(card.plan.durationValue).toBe(3);
    expect(card.plan.policy.monthlyOrderLimit).toBe(8);
    expect(card.usage).toEqual({
      monthlyConsumption: 6800,
      monthlyOrderCount: 2,
    });
  });
});
