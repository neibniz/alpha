// @ts-nocheck
import { describe, expect, test } from "bun:test";
import { isEmailDomainAllowed, renderAppSubLink } from "./index.js";

describe("shared utility helpers", () => {
  test("checks email domains against normalized public config arrays", () => {
    expect(
      isEmailDomainAllowed("User@Example.com", ["example.com", "alpha.dev"])
    ).toBe(true);
    expect(isEmailDomainAllowed("user@blocked.test", ["example.com"])).toBe(
      false
    );
  });

  test("rejects malformed emails and empty allow lists", () => {
    expect(isEmailDomainAllowed("not-an-email", ["example.com"])).toBe(false);
    expect(isEmailDomainAllowed("user@example.com", [])).toBe(false);
  });

  test("renders app subscription link JSON expressions with bare identifiers", () => {
    const result = renderAppSubLink(
      '${JSON.stringify({url:url,name:name,label:"url name"})}',
      {
        name: "Alpha Node",
        url: "https://sub.example/link?token=abc",
      }
    );

    expect(JSON.parse(result)).toEqual({
      label: "url name",
      name: "Alpha Node",
      url: "https://sub.example/link?token=abc",
    });
  });

  test("renders encoded app subscription link JSON expressions in one pass", () => {
    const result = renderAppSubLink(
      "${encodeURIComponent(JSON.stringify({server:url,tag:name}))}",
      {
        name: "Alpha Node",
        url: "https://sub.example/link?token=abc",
      }
    );

    expect(JSON.parse(decodeURIComponent(result))).toEqual({
      server: "https://sub.example/link?token=abc",
      tag: "Alpha Node",
    });
  });
});
