type ConversionType =
  | "centsToDollars"
  | "dollarsToCents"
  | "bitsToMb"
  | "mbToBits"
  | "bytesToGb"
  | "gbToBytes";

const conversionConfig: Record<
  ConversionType,
  { convert: (value: number) => number; precision: number }
> = {
  centsToDollars: { convert: (value) => value / 100, precision: 2 },
  dollarsToCents: { convert: (value) => value * 100, precision: 0 },
  bitsToMb: { convert: (value) => value / 1024 ** 2, precision: 2 },
  mbToBits: { convert: (value) => value * 1024 ** 2, precision: 0 },
  bytesToGb: { convert: (value) => value / 1024 ** 3, precision: 2 },
  gbToBytes: { convert: (value) => value * 1024 ** 3, precision: 0 },
};

export function unitConversion(type: ConversionType, value?: number | string) {
  if (!value) return 0;

  const config = conversionConfig[type];
  if (!config) throw new Error("Invalid conversion type");
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue)) return 0;
  return Number(config.convert(numericValue).toFixed(config.precision));
}

export function evaluateWithPrecision(value: number) {
  if (!Number.isFinite(value)) return 0;
  return Number(value.toFixed(2));
}
