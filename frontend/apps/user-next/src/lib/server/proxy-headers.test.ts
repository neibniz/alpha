import { describe, expect, test } from "bun:test";
import { upstreamRequestHeaders } from "./proxy-headers";

describe("upstreamRequestHeaders", () => {
  const incoming = new Headers({
    "user-agent": "Alpha Browser",
    "x-forwarded-for": "203.0.113.20",
    "x-forwarded-host": "attacker.example",
    "x-forwarded-proto": "https",
    "x-original-forwarded-for": "198.51.100.10",
    "x-real-ip": "203.0.113.20",
  });

  test("forwards the user agent without trusting proxy metadata", () => {
    const headers = upstreamRequestHeaders(incoming, {
      Accept: "application/json",
    });

    expect(headers.get("accept")).toBe("application/json");
    expect(headers.get("user-agent")).toBe("Alpha Browser");
    expect(headers.get("x-forwarded-for")).toBe(null);
    expect(headers.get("x-real-ip")).toBe(null);
  });

  test("forwards only the client metadata produced by a trusted proxy", () => {
    const headers = upstreamRequestHeaders(incoming, undefined, true);

    expect(headers.get("x-forwarded-for")).toBe("203.0.113.20");
    expect(headers.get("x-forwarded-proto")).toBe("https");
    expect(headers.get("x-real-ip")).toBe("203.0.113.20");
    expect(headers.get("x-forwarded-host")).toBe(null);
    expect(headers.get("x-original-forwarded-for")).toBe(null);
  });
});
