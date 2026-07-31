import { describe, expect, test } from "vitest";
import {
  buildProductPortfolioSummary,
  filterProducts,
  getProductLifecycle,
  getVisibleProductMatrixRows,
  isProductLowStock,
  moveProduct,
  normalizeMonthlyPrice,
} from "./product-data";

function product(
  overrides: Partial<API.SubscribeItem> = {}
): API.SubscribeItem {
  return {
    id: 1,
    name: "进阶月付",
    language: "",
    unit_price: 5900,
    unit_time: "Month",
    replacement: 0,
    inventory: -1,
    traffic: 500 * 1024 ** 3,
    speed_limit: 500_000_000,
    device_limit: 5,
    quota: 0,
    show: true,
    user_visible: true,
    sell: true,
    allow_renewal: true,
    reset_cycle: 2,
    sort: 1,
    created_at: 0,
    updated_at: 0,
    sold: 10,
    ...overrides,
  };
}

describe("product portfolio data", () => {
  test("derives lifecycle from sales and visibility state", () => {
    expect(getProductLifecycle(product())).toBe("selling");
    expect(getProductLifecycle(product({ sell: false }))).toBe("draft");
    expect(
      getProductLifecycle(
        product({ sell: false, show: false, user_visible: false })
      )
    ).toBe("hidden");
  });

  test("summarizes active subscriptions and monthly list revenue", () => {
    const summary = buildProductPortfolioSummary([
      product(),
      product({
        id: 2,
        unit_price: 12_000,
        unit_time: "Year",
        sold: 12,
        inventory: 10,
      }),
    ]);
    expect(summary).toEqual({
      total: 2,
      selling: 2,
      lowStock: 1,
      subscriptions: 22,
      monthlyListRevenue: 71_000,
    });
  });

  test("normalizes recurring prices to a monthly amount", () => {
    expect(normalizeMonthlyPrice(120_000, "Year")).toBe(10_000);
    expect(normalizeMonthlyPrice(100, "Day")).toBe(3000);
    expect(normalizeMonthlyPrice(100, "NoLimit")).toBe(0);
  });

  test("filters search and language without mutating products", () => {
    const products = [
      product(),
      product({ id: 2, name: "Global", language: "en-US" }),
    ];
    expect(
      filterProducts(products, "global", "en-US").map((item) => item.id)
    ).toEqual([2]);
    expect(products[0]?.id).toBe(1);
  });

  test("shows only comparison rows whose values differ", () => {
    const rows = getVisibleProductMatrixRows(
      [product(), product({ id: 2, device_limit: 8 })],
      true
    );
    expect(rows).toEqual(["devices"]);
  });

  test("moves products and rewrites stable sort values", () => {
    const result = moveProduct(
      [product(), product({ id: 2, sort: 2 }), product({ id: 3, sort: 3 })],
      2,
      -1
    );
    expect(result.map((item) => item.id)).toEqual([2, 1, 3]);
    expect(result.map((item) => item.sort)).toEqual([1, 2, 3]);
  });

  test("uses a bounded low-stock threshold and preserves unlimited stock", () => {
    expect(isProductLowStock(product({ inventory: 20 }))).toBe(true);
    expect(isProductLowStock(product({ inventory: 21 }))).toBe(false);
    expect(isProductLowStock(product({ inventory: -1 }))).toBe(false);
  });
});
