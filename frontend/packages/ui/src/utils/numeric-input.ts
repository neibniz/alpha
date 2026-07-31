import type { HTMLInputTypeAttribute } from "react";

export function isIntegerNumberInput(
  type: HTMLInputTypeAttribute | undefined,
  step: number | string | undefined
) {
  return type === "number" && Number(step) === 1;
}

export function normalizeIntegerInputValue(
  value: string,
  allowNegative: boolean
) {
  const trimmed = value.trim();
  if (!trimmed) return "";
  if (allowNegative && trimmed === "-") return trimmed;
  if (/^-?\d+$/.test(trimmed)) {
    return allowNegative || !trimmed.startsWith("-") ? trimmed : "";
  }

  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || (!allowNegative && parsed < 0)) return "";
  return String(Math.trunc(parsed));
}

export function isDisallowedIntegerKey(key: string, allowNegative: boolean) {
  return (
    key === "." ||
    key === "," ||
    key === "e" ||
    key === "E" ||
    key === "+" ||
    (!allowNegative && key === "-")
  );
}
