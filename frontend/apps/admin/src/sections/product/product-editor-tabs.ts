export const PRODUCT_EDITOR_FIELDS = {
  basic: [
    "name",
    "language",
    "traffic",
    "speed_limit",
    "device_limit",
    "inventory",
    "quota",
    "description",
  ],
  pricing: [
    "unit_price",
    "unit_time",
    "replacement",
    "reset_cycle",
    "discount",
    "deduction_ratio",
    "allow_renewal",
    "allow_reset_traffic",
    "renewal_reset",
    "show_original_price",
    "allow_deduction",
  ],
  storefront: [
    "show",
    "user_visible",
    "sell",
    "display_template",
    "badge",
    "min_quantity",
    "max_quantity",
    "sale_starts_at",
    "sale_ends_at",
  ],
  servers: ["node_tags", "nodes"],
} as const;

export function productErrorTab(fields: string[]): string {
  return (
    Object.entries(PRODUCT_EDITOR_FIELDS).find(([, names]) =>
      fields.some((field) =>
        (names as readonly string[]).includes(field.split(".")[0]!)
      )
    )?.[0] ?? "basic"
  );
}
