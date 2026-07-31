import { describe, expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { Markdown } from "./markdown";

describe("Markdown", () => {
  test("sanitizes untrusted HTML and unsafe URLs", () => {
    const html = renderToStaticMarkup(
      <Markdown>{`<div onclick="alert('xss')"><script>alert('xss')</script></div>

[unsafe](javascript:alert('xss'))`}</Markdown>
    );

    expect(html).not.toContain("onclick");
    expect(html).not.toContain("<script");
    expect(html).not.toContain("javascript:");
  });

  test("renders math after sanitization", () => {
    const html = renderToStaticMarkup(<Markdown>{"$E = mc^2$"}</Markdown>);

    expect(html).toContain("katex");
  });
});
