export type ProductLifecycle = "selling" | "draft" | "hidden";

export type ProductMatrixRowKey =
  | "positioning"
  | "traffic"
  | "speed"
  | "devices"
  | "reset"
  | "renewal"
  | "visibility"
  | "inventory"
  | "subscriptions";

export const PRODUCT_MATRIX_ROWS: ProductMatrixRowKey[] = [
  "positioning",
  "traffic",
  "speed",
  "devices",
  "reset",
  "renewal",
  "visibility",
  "inventory",
  "subscriptions",
];

const GIBIBYTE = 1024 ** 3;
const TEBIBYTE = 1024 ** 4;

export function getProductLifecycle(
  product: Pick<API.SubscribeItem, "sell" | "show" | "user_visible">
): ProductLifecycle {
  if (product.sell) return "selling";
  if (product.show || product.user_visible) return "draft";
  return "hidden";
}

export function isProductLowStock(
  product: Pick<API.SubscribeItem, "inventory">
) {
  return product.inventory >= 0 && product.inventory <= 20;
}

export function normalizeMonthlyPrice(unitPrice: number, unitTime: string) {
  const price = Math.max(0, Number(unitPrice) || 0);
  switch (unitTime.toLocaleLowerCase()) {
    case "year":
      return Math.round(price / 12);
    case "day":
      return price * 30;
    case "hour":
      return price * 24 * 30;
    case "minute":
      return price * 60 * 24 * 30;
    case "nolimit":
      return 0;
    default:
      return price;
  }
}

export function buildProductPortfolioSummary(products: API.SubscribeItem[]) {
  let selling = 0;
  let lowStock = 0;
  let subscriptions = 0;
  let monthlyListRevenue = 0;

  for (const product of products) {
    if (getProductLifecycle(product) === "selling") selling += 1;
    if (isProductLowStock(product)) lowStock += 1;
    subscriptions += Math.max(0, product.sold || 0);
    monthlyListRevenue +=
      normalizeMonthlyPrice(product.unit_price, product.unit_time) *
      Math.max(0, product.sold || 0);
  }

  return {
    total: products.length,
    selling,
    lowStock,
    subscriptions,
    monthlyListRevenue,
  };
}

export function filterProducts(
  products: API.SubscribeItem[],
  search: string,
  language: string
) {
  const query = search.trim().toLocaleLowerCase();
  return products.filter((product) => {
    const matchesLanguage =
      language === "all" ||
      (language === "default" && !product.language) ||
      product.language === language;
    if (!matchesLanguage) return false;
    if (!query) return true;
    return [
      product.name,
      product.language,
      String(product.id),
      getProductLifecycle(product),
    ].some((value) => value.toLocaleLowerCase().includes(query));
  });
}

export function getProductPositioning(product: API.SubscribeItem) {
  const name = product.name.toLocaleLowerCase();
  if (name.includes("团队") || name.includes("team")) return "team";
  if (
    ["day", "hour", "minute"].includes(product.unit_time.toLocaleLowerCase())
  ) {
    return "trial";
  }
  if (product.traffic === 0) return "unlimited";
  if (product.traffic >= TEBIBYTE || product.speed_limit >= 1_000_000_000) {
    return "performance";
  }
  if (product.traffic >= 500 * GIBIBYTE) return "heavy";
  return "light";
}

function comparisonValue(product: API.SubscribeItem, row: ProductMatrixRowKey) {
  switch (row) {
    case "positioning":
      return getProductPositioning(product);
    case "traffic":
      return String(product.traffic || 0);
    case "speed":
      return String(product.speed_limit || 0);
    case "devices":
      return String(product.device_limit || 0);
    case "reset":
      return String(product.reset_cycle || 0);
    case "renewal":
      return String(Boolean(product.allow_renewal));
    case "visibility":
      return `${Number(product.show)}${Number(product.user_visible)}${Number(product.sell)}`;
    case "inventory":
      return String(product.inventory);
    case "subscriptions":
      return String(product.sold || 0);
  }
}

export function getVisibleProductMatrixRows(
  products: API.SubscribeItem[],
  onlyDifferences: boolean
) {
  if (!onlyDifferences || products.length < 2) return PRODUCT_MATRIX_ROWS;
  return PRODUCT_MATRIX_ROWS.filter((row) => {
    const values = new Set(
      products.map((product) => comparisonValue(product, row))
    );
    return values.size > 1;
  });
}

export function sortProducts(products: API.SubscribeItem[]) {
  return products.slice().sort((left, right) => {
    if (left.sort !== right.sort) return left.sort - right.sort;
    return left.id - right.id;
  });
}

export function moveProduct(
  products: API.SubscribeItem[],
  productId: number,
  direction: -1 | 1
) {
  const ordered = sortProducts(products);
  const sourceIndex = ordered.findIndex((product) => product.id === productId);
  const targetIndex = sourceIndex + direction;
  if (sourceIndex < 0 || targetIndex < 0 || targetIndex >= ordered.length) {
    return ordered;
  }

  const [moved] = ordered.splice(sourceIndex, 1);
  if (!moved) return ordered;
  ordered.splice(targetIndex, 0, moved);
  const existingSorts = ordered
    .map((product) => product.sort)
    .filter((value) => Number.isFinite(value));
  const baseSort = existingSorts.length > 0 ? Math.min(...existingSorts) : 0;
  return ordered.map((product, index) => ({
    ...product,
    sort: baseSort + index,
  }));
}
