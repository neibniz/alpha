import { describe, expect, it } from "vitest";
import {
  MASKED_LOG_VISIBILITY,
  sensitiveLogQuery,
} from "./sensitive-log-toggle";

describe("sensitive log query", () => {
  it("omits reveal parameters in the default masked state", () => {
    expect(sensitiveLogQuery(MASKED_LOG_VISIBILITY)).toEqual({});
    expect(sensitiveLogQuery({ enabled: true })).toEqual({});
  });

  it("sends an audited reason only while complete information is enabled", () => {
    expect(
      sensitiveLogQuery({ enabled: true, reason: "security_incident" })
    ).toEqual({
      reveal_sensitive: true,
      reveal_reason: "security_incident",
    });
  });
});
