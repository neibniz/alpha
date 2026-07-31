import { describe, expect, it } from "vitest";
import { createGithubSchema } from "./github-schema";

const schema = createGithubSchema({
  clientIdRequired: "client-id-required",
  clientSecretRequired: "client-secret-required",
  callbackUrlRequired: "callback-required",
  callbackUrlInvalid: "callback-invalid",
});

const emptyConfig = {
  enabled: false,
  client_id: "",
  client_secret: "",
  client_secret_configured: false,
  redirect_url: "",
};

describe("GitHub OAuth admin schema", () => {
  it("allows an empty configuration while GitHub OAuth is disabled", () => {
    expect(schema.safeParse(emptyConfig).success).toBe(true);
  });

  it("requires all credentials before GitHub OAuth can be enabled", () => {
    const result = schema.safeParse({ ...emptyConfig, enabled: true });

    expect(result.success).toBe(false);
    if (result.success) return;
    expect(result.error.issues.map((issue) => issue.path[0])).toEqual([
      "client_id",
      "client_secret",
      "redirect_url",
    ]);
  });

  it("keeps an existing secret and accepts a frontend origin root", () => {
    expect(
      schema.safeParse({
        ...emptyConfig,
        enabled: true,
        client_id: "Iv1.client",
        client_secret_configured: true,
        redirect_url: "https://user.example.com/",
      }).success
    ).toBe(true);
  });

  it("rejects callback paths, query strings, and fragments", () => {
    for (const redirect_url of [
      "https://user.example.com/oauth",
      "https://user.example.com/?source=github",
      "https://user.example.com/#/oauth",
    ]) {
      const result = schema.safeParse({
        ...emptyConfig,
        enabled: true,
        client_id: "Iv1.client",
        client_secret: "secret",
        redirect_url,
      });
      expect(result.success).toBe(false);
    }
  });
});
