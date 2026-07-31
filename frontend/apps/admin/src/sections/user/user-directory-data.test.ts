import { describe, expect, test } from "vitest";
import {
  buildUserPageSummary,
  getUserLifecycle,
  getUserWalletTotal,
  maskUserIdentifier,
} from "./user-directory-data";

function user(overrides: Partial<API.UserListItem> = {}): API.UserListItem {
  return {
    id: 1,
    balance: 1200,
    commission: 300,
    gift_amount: 500,
    member_expired_at: 0,
    is_member: false,
    refer_code: "ALPHA",
    referer_id: 0,
    enable: true,
    primary_auth_method: {
      auth_type: "email",
      auth_identifier: "member@example.com",
      verified: true,
    },
    created_at: 0,
    ...overrides,
  };
}

describe("user directory data", () => {
  test("masks email, mobile, and provider identifiers", () => {
    expect(maskUserIdentifier("member@example.com", "email")).toBe(
      "me***@example.com"
    );
    expect(maskUserIdentifier("8613800138000", "mobile")).toBe("861****8000");
    expect(maskUserIdentifier("github-open-id-123456", "github")).toBe(
      "gi***56"
    );
  });

  test("derives mutually exclusive lifecycle states", () => {
    expect(getUserLifecycle(user())).toBe("enabled");
    expect(getUserLifecycle(user({ enable: false }))).toBe("disabled");
    expect(getUserLifecycle(user({ deleted_at: 1 }))).toBe("deleted");
  });

  test("summarizes the current page without presenting it as global data", () => {
    expect(
      buildUserPageSummary(
        [
          user(),
          user({ id: 2, enable: false, is_member: true }),
          user({ id: 3, deleted_at: 1, is_member: true }),
        ],
        28
      )
    ).toEqual({
      matchingTotal: 28,
      pageTotal: 3,
      enabled: 1,
      members: 1,
      attention: 2,
    });
  });

  test("combines wallet balances while ignoring invalid negative values", () => {
    expect(getUserWalletTotal(user())).toBe(2000);
    expect(
      getUserWalletTotal(
        user({ balance: -100, commission: 200, gift_amount: 0 })
      )
    ).toBe(200);
  });
});
