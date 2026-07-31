type RequestBodyWithId = {
  request_id?: string;
};

export function createRequestId(): string {
  const cryptoApi = globalThis.crypto;
  if (!cryptoApi?.getRandomValues) {
    throw new Error("Secure random number generation is unavailable");
  }

  if (typeof cryptoApi.randomUUID === "function") {
    return cryptoApi.randomUUID();
  }

  const bytes = cryptoApi.getRandomValues(new Uint8Array(16));
  // biome-ignore lint/suspicious/noBitwiseOperators: RFC 4122 requires masking the version bits.
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  // biome-ignore lint/suspicious/noBitwiseOperators: RFC 4122 requires masking the variant bits.
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0"));

  return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex
    .slice(6, 8)
    .join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10).join("")}`;
}

export function withRequestId<T extends RequestBodyWithId>(
  body: T
): T & { request_id: string } {
  const requestId = body.request_id?.trim();
  return {
    ...body,
    request_id: requestId || createRequestId(),
  };
}
