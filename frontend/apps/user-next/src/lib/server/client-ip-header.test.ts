import { describe, expect, test } from "bun:test";
import {
  attachEdgeClientIPHeader,
  EDGE_CLIENT_IP_HEADER,
  resolveEdgeClientIP,
} from "./client-ip-header";

function withClientIPSource(source: string, run: () => void) {
  const original = process.env.ALPHA_CLIENT_IP_SOURCE;
  process.env.ALPHA_CLIENT_IP_SOURCE = source;
  try {
    run();
  } finally {
    if (original === undefined) delete process.env.ALPHA_CLIENT_IP_SOURCE;
    else process.env.ALPHA_CLIENT_IP_SOURCE = original;
  }
}

describe("edge client IP header", () => {
  const incoming = new Headers({
    "cf-connecting-ip": "203.0.113.42",
    "x-forwarded-for": "198.51.100.8, 192.0.2.2",
    "x-real-ip": "192.0.2.9",
  });

  test("uses the IP source selected for the deployment", () => {
    expect(resolveEdgeClientIP(incoming, "cloudflare")).toBe("203.0.113.42");
    expect(resolveEdgeClientIP(incoming, "vercel")).toBe("198.51.100.8");
    expect(resolveEdgeClientIP(incoming, "forwarded")).toBe("192.0.2.9");
    expect(resolveEdgeClientIP(incoming, "auto")).toBe("203.0.113.42");
  });

  test("overwrites a caller-supplied edge header", () => {
    withClientIPSource("cloudflare", () => {
      const destination = new Headers({
        [EDGE_CLIENT_IP_HEADER]: "198.51.100.200",
      });

      attachEdgeClientIPHeader(destination, incoming);

      expect(destination.get(EDGE_CLIENT_IP_HEADER)).toBe("203.0.113.42");
    });
  });

  test("can be disabled", () => {
    withClientIPSource("none", () => {
      const destination = new Headers({
        [EDGE_CLIENT_IP_HEADER]: "198.51.100.200",
      });

      attachEdgeClientIPHeader(destination, incoming);

      expect(destination.has(EDGE_CLIENT_IP_HEADER)).toBe(false);
    });
  });

  test("rejects malformed addresses", () => {
    const malformed = new Headers({
      "x-forwarded-for": "unknown, 198.51.100.8",
    });
    expect(resolveEdgeClientIP(malformed, "vercel")).toBe(undefined);
  });
});
