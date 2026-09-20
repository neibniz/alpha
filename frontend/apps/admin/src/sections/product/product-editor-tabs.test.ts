import { expect, test } from "vitest";
import { productErrorTab } from "./product-editor-tabs";

test("opens the tab containing the invalid product field", () => {
  expect(productErrorTab(["unit_price"])).toBe("pricing");
  expect(productErrorTab(["discount.0.discount"])).toBe("pricing");
  expect(productErrorTab(["sale_ends_at"])).toBe("storefront");
  expect(productErrorTab(["nodes"])).toBe("servers");
  expect(productErrorTab(["name", "nodes"])).toBe("basic");
});
