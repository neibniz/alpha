import { describe, expect, it } from "vitest";
import { subscribeSchema } from "./schema";

describe("subscription template schema", () => {
  const proxyFields = subscribeSchema.Proxies.items.properties;

  it("matches the fields exposed by the server template adapter", () => {
    expect(subscribeSchema).toHaveProperty("OutputFormat");
    expect(subscribeSchema).toHaveProperty("Params");
    expect(proxyFields).toHaveProperty("Network");
    expect(proxyFields).toHaveProperty("GrpcUserAgent");
    expect(proxyFields).toHaveProperty("GrpcPermitWithoutStream");
    expect(proxyFields).toHaveProperty("GrpcInitialWindowsSize");
    expect(proxyFields).toHaveProperty("HysteriaUDPIdleTimeout");
    expect(proxyFields).toHaveProperty("HysteriaMasquerade");
    expect(proxyFields).toHaveProperty("Ratio");
    expect(proxyFields).toHaveProperty("CertMode");
  });

  it("does not advertise fields absent from the server template data", () => {
    expect(proxyFields).not.toHaveProperty("Enable");
    expect(proxyFields).not.toHaveProperty("UUID");
    expect(proxyFields).not.toHaveProperty("AlterId");
    expect(proxyFields).not.toHaveProperty("Password");
  });
});
