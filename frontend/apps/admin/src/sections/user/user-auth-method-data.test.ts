import { describe, expect, it } from "vitest";
import {
  buildUserAuthMethodData,
  preferredAuthIdentifier,
} from "./user-auth-method-data";

describe("admin user auth method data", () => {
  it("classifies email and excludes removed sign-in methods", () => {
    const methods = [
      { auth_type: "github", auth_identifier: "octo", verified: true },
      {
        auth_type: "email",
        auth_identifier: "first@example.com",
        verified: true,
      },
      { auth_type: "mobile", auth_identifier: "+15550000", verified: false },
      { auth_type: "device", auth_identifier: "legacy-device", verified: true },
      {
        auth_type: "email",
        auth_identifier: "second@example.com",
        verified: false,
      },
    ] as API.UserAuthMethod[];

    const result = buildUserAuthMethodData(methods);

    expect(result.isEmailExists).toBe(true);
    expect(result.emailMethod?.auth_identifier).toBe("first@example.com");
    expect(result.defaultEmailMethod).toEqual(methods[1]);
    expect(result.otherMethods.map((method) => method.auth_type)).toEqual([
      "github",
    ]);
  });

  it("provides a stable default email method when missing", () => {
    const result = buildUserAuthMethodData([
      { auth_type: "github", auth_identifier: "octo", verified: true },
    ] as API.UserAuthMethod[]);

    expect(result.isEmailExists).toBe(false);
    expect(result.defaultEmailMethod).toEqual({
      auth_type: "email",
      auth_identifier: "",
      verified: false,
    });
    expect(result.otherMethods).toHaveLength(1);
  });

  it("prefers a non-empty email identifier and falls back to the first method", () => {
    expect(
      preferredAuthIdentifier([
        { auth_type: "github", auth_identifier: "octo", verified: true },
        {
          auth_type: "email",
          auth_identifier: "mail@example.com",
          verified: true,
        },
      ] as API.UserAuthMethod[])
    ).toBe("mail@example.com");

    expect(
      preferredAuthIdentifier([
        { auth_type: "github", auth_identifier: "octo", verified: true },
        { auth_type: "email", auth_identifier: "", verified: true },
      ] as API.UserAuthMethod[])
    ).toBe("octo");
  });
});
