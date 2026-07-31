export type ProductFeatureType = "default" | "success" | "destructive";

export type ProductFeature = {
  icon: string;
  label: string;
  type: ProductFeatureType;
};

export type ProductDescription = {
  description: string;
  features: ProductFeature[];
};

const FEATURE_TYPES = new Set<ProductFeatureType>([
  "default",
  "success",
  "destructive",
]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeFeature(value: unknown): ProductFeature | undefined {
  if (!isRecord(value)) return;
  const label = typeof value.label === "string" ? value.label : "";
  const icon = typeof value.icon === "string" ? value.icon : "";
  const type = FEATURE_TYPES.has(value.type as ProductFeatureType)
    ? (value.type as ProductFeatureType)
    : "default";
  if (!(label || icon)) return;
  return { icon, label, type };
}

export function parseProductDescription(value?: string): ProductDescription {
  if (!value) return { description: "", features: [] };
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!isRecord(parsed)) {
      return { description: value, features: [] };
    }
    return {
      description:
        typeof parsed.description === "string" ? parsed.description : "",
      features: Array.isArray(parsed.features)
        ? parsed.features
            .map(normalizeFeature)
            .filter((item): item is ProductFeature => Boolean(item))
        : [],
    };
  } catch {
    return { description: value, features: [] };
  }
}

export function serializeProductDescription(value: ProductDescription) {
  return JSON.stringify({
    description: value.description.trim(),
    features: value.features
      .map((feature) => ({
        type: feature.type,
        icon: feature.icon.trim(),
        label: feature.label.trim(),
      }))
      .filter((feature) => feature.label || feature.icon),
  });
}
