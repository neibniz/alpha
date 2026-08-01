import { describe, expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { Switch } from "./switch";

describe("Switch", () => {
  test("keeps an intrinsic width inside full-width form fields", () => {
    const html = renderToStaticMarkup(
      <div className="[&>*]:w-full">
        <Switch aria-label="Example setting" />
      </div>
    );

    expect(html).toContain("w-8");
    expect(html).toContain("min-w-8");
    expect(html).toContain("max-w-8");
    expect(html).toContain('role="switch"');
  });
});
