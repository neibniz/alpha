import { describe, expect, test } from "bun:test";
import { githubOAuthCallbackUrl } from "./oauth-callback";

describe("githubOAuthCallbackUrl", () => {
  test("uses the canonical login callback route", () => {
    expect(githubOAuthCallbackUrl("https://user.example/", "login")).toBe(
      "https://user.example/oauth/github/callback"
    );
  });

  test("uses the canonical account binding callback route", () => {
    expect(githubOAuthCallbackUrl("https://user.example", "bind")).toBe(
      "https://user.example/bind/github/callback"
    );
  });

  test("discards any path from the configured public origin", () => {
    expect(githubOAuthCallbackUrl("https://user.example/base", "login")).toBe(
      "https://user.example/oauth/github/callback"
    );
  });
});
