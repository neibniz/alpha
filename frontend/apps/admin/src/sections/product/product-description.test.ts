import { describe, expect, test } from "vitest";
import {
  parseProductDescription,
  serializeProductDescription,
} from "./product-description";

describe("product description", () => {
  test("parses structured descriptions and normalizes feature types", () => {
    expect(
      parseProductDescription(
        JSON.stringify({
          description: "Fast plan",
          features: [
            { label: "Priority", icon: "uil:bolt", type: "success" },
            { label: "Fallback", type: "other" },
          ],
        })
      )
    ).toEqual({
      description: "Fast plan",
      features: [
        { label: "Priority", icon: "uil:bolt", type: "success" },
        { label: "Fallback", icon: "", type: "default" },
      ],
    });
  });

  test("keeps legacy plain text editable", () => {
    expect(parseProductDescription("Legacy copy")).toEqual({
      description: "Legacy copy",
      features: [],
    });
  });

  test("serializes trimmed structured content", () => {
    expect(
      JSON.parse(
        serializeProductDescription({
          description: " Intro ",
          features: [
            { label: " Fast ", icon: " uil:bolt ", type: "success" },
            { label: " ", icon: " ", type: "default" },
          ],
        })
      )
    ).toEqual({
      description: "Intro",
      features: [{ label: "Fast", icon: "uil:bolt", type: "success" }],
    });
  });
});
