import { differenceInMilliseconds, intlFormat } from "date-fns";

export function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${(bytes / k ** i).toFixed(2)} ${sizes[i]}`;
}

export function formatCurrencyAmount(cents: number, symbol = "") {
  const amount = cents / 100;
  const sign = amount < 0 ? "-" : "";
  return `${sign}${symbol}${Math.abs(amount).toFixed(2)}`;
}

function readTimeZone(): string {
  try {
    return globalThis.window?.localStorage?.getItem("timezone") || "UTC";
  } catch {
    return "UTC";
  }
}

export function formatDate(date?: Date | number, showTime = true) {
  if (!date) return;
  // Convert Unix seconds to milliseconds automatically
  const finalDate =
    typeof date === "number" && date < 100_000_000_000 ? date * 1000 : date;
  const timeZone = readTimeZone();
  const options: Intl.DateTimeFormatOptions = {
    year: "numeric",
    month: "numeric",
    day: "numeric",
    hour: showTime ? "numeric" : undefined,
    minute: showTime ? "numeric" : undefined,
    second: showTime ? "numeric" : undefined,
    hour12: false,
    timeZone,
  };
  try {
    return intlFormat(finalDate, options);
  } catch (error) {
    if (!(error instanceof RangeError) || timeZone === "UTC") throw error;
    return intlFormat(finalDate, { ...options, timeZone: "UTC" });
  }
}

export function differenceInDays(
  dateLeft: Date | number,
  dateRight: Date | number
) {
  const finalDateLeft =
    typeof dateLeft === "number" && dateLeft < 100_000_000_000
      ? dateLeft * 1000
      : dateLeft;
  const finalDateRight =
    typeof dateRight === "number" && dateRight < 100_000_000_000
      ? dateRight * 1000
      : dateRight;

  const diffInMs = differenceInMilliseconds(finalDateLeft, finalDateRight);
  const diffInDays = diffInMs / (1000 * 60 * 60 * 24);

  if (Math.abs(diffInDays) >= 1) return Number(diffInDays.toFixed(0));
  return Number(diffInDays.toFixed(2));
}
