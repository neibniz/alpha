type RequestIdCrypto = {
  getRandomValues: (array: Uint8Array) => Uint8Array;
  randomUUID?: () => string;
};

function bytesToUuid(bytes: Uint8Array) {
  const hex = Array.from(bytes, (value) =>
    value.toString(16).padStart(2, "0")
  ).join("");
  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20),
  ].join("-");
}

export function createOrderRequestId(
  source: RequestIdCrypto | undefined = globalThis.crypto
) {
  if (typeof source?.randomUUID === "function") {
    return source.randomUUID();
  }
  if (typeof source?.getRandomValues !== "function") {
    throw new Error("当前浏览器无法生成安全的订单请求标识");
  }

  const bytes = source.getRandomValues(new Uint8Array(16));
  bytes[6] = ((bytes[6] ?? 0) % 16) + 64;
  bytes[8] = ((bytes[8] ?? 0) % 64) + 128;
  return bytesToUuid(bytes);
}
