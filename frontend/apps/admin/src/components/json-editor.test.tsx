/** @vitest-environment jsdom */
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { JSONEditor } from "@workspace/ui/composed/editor/json";
import { afterEach, expect, test, vi } from "vitest";

vi.mock("@workspace/ui/composed/editor/monaco-editor", () => ({
  MonacoEditor: ({
    value,
    onChange,
  }: {
    value?: string;
    onChange?: (value: string) => void;
  }) => (
    <textarea
      aria-label="JSON"
      value={value}
      onChange={(event) => onChange?.(event.target.value)}
    />
  ),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (_key: string, fallback: string) => fallback }),
}));
afterEach(cleanup);

test.each([
  "",
  "{}",
])("keeps string values as strings, including an empty initial value: %j", (value) => {
  const onChange = vi.fn();
  render(<JSONEditor value={value} onChange={onChange} />);
  fireEvent.change(screen.getByRole("textbox", { name: "JSON" }), {
    target: { value: '{"website":"https://example.org"}' },
  });
  expect(onChange).toHaveBeenLastCalledWith(
    '{"website":"https://example.org"}'
  );
});
