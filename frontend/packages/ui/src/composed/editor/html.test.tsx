import { describe, expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { HTMLPreview } from "./html";

describe("HTMLPreview", () => {
  test("isolates preview content without script permissions", () => {
    const html = renderToStaticMarkup(
      <HTMLPreview
        value={'<script>parent.location="https://invalid"</script>'}
      />
    );

    expect(html).toContain('sandbox=""');
    expect(html).not.toContain("allow-scripts");
  });
});
