/** @vitest-environment jsdom */
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { buildOutboundConfig, configToFormValues } from "./config";
import { JsonObjectEditor } from "./json-form-controls";

beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn();
  Element.prototype.hasPointerCapture = vi.fn(() => false);
  Element.prototype.setPointerCapture = vi.fn();
  Element.prototype.releasePointerCapture = vi.fn();
});
afterEach(cleanup);

function Harness({ initial = "{}" }: { initial?: string }) {
  const [value, setValue] = useState(initial);
  const [renders, setRenders] = useState(0);
  return (
    <>
      <JsonObjectEditor onChange={setValue} value={value} />
      <output data-testid="value">{value}</output>
      <button type="button" onClick={() => setRenders(renders + 1)}>
        Rerender {renders}
      </button>
    </>
  );
}

function serialized() {
  return screen.getByTestId("value").textContent || "";
}

function selectType(name: string) {
  fireEvent.keyDown(screen.getByRole("combobox", { name: "字段类型" }), {
    key: " ",
  });
  fireEvent.click(screen.getByRole("option", { name }));
}

test("an empty field can switch to JSON without reverting to text", () => {
  render(<Harness />);
  fireEvent.click(screen.getByRole("button", { name: "添加字段" }));
  selectType("JSON");
  expect(screen.getByRole("combobox", { name: "字段类型" }).textContent).toBe(
    "JSON"
  );
  expect(JSON.parse(serialized())).toEqual({ key_1: {} });
  expect(
    (screen.getByRole("textbox", { name: "key_1 JSON" }) as HTMLTextAreaElement)
      .value
  ).toBe("{}");
});

test("partial JSON remains editable and cannot silently become a valid string", () => {
  render(<Harness initial='{"finalRules":[]}' />);
  const input = screen.getByRole("textbox", { name: "finalRules JSON" });
  fireEvent.change(input, { target: { value: '[\n  {"action":' } });
  expect(() => JSON.parse(serialized())).toThrow();
  expect(screen.getByText("请输入完整、有效的 JSON。")).toBeTruthy();
  expect(screen.getByRole("textbox", { name: "finalRules JSON" })).toBe(input);
  fireEvent.click(screen.getByRole("button", { name: "Rerender 0" }));
  expect((input as HTMLTextAreaElement).value).toBe('[\n  {"action":');
  const rules = [
    { action: "allow", network: "tcp", ip: ["127.0.0.1/32"], port: "5080" },
  ];
  const text = JSON.stringify(rules, null, 4);
  fireEvent.change(input, { target: { value: text } });
  expect(JSON.parse(serialized())).toEqual({ finalRules: rules });
  expect((input as HTMLTextAreaElement).value).toBe(text);
  expect(screen.queryByText("请输入完整、有效的 JSON。")).toBeNull();
});

test("renaming fields does not remount the focused input or overwrite duplicates", () => {
  render(<Harness initial='{"first":"a","second":"b"}' />);
  const input = screen.getAllByRole("textbox", { name: "字段名" })[0];
  if (!input) throw new Error("Missing field name input");
  input.focus();
  for (const name of ["f", "fi", "finalRules"]) {
    fireEvent.change(input, { target: { value: name } });
    expect(document.activeElement).toBe(input);
  }
  fireEvent.change(input, { target: { value: "second" } });
  expect(screen.getAllByText("字段名不能重复。")).toHaveLength(2);
  expect(() => JSON.parse(serialized())).toThrow();
  fireEvent.change(input, { target: { value: "unique" } });
  expect(JSON.parse(serialized())).toEqual({ unique: "a", second: "b" });
});

test("preserves zero, false, null and explicitly selected JSON scalar types", () => {
  render(<Harness initial='{"zero":0,"disabled":false,"empty":null}' />);
  const input = screen.getByRole("textbox", { name: "zero 值" });
  expect((input as HTMLInputElement).value).toBe("0");
  fireEvent.change(input, { target: { value: "1e2" } });
  expect(JSON.parse(serialized())).toEqual({
    zero: 100,
    disabled: false,
    empty: null,
  });
  fireEvent.change(screen.getByRole("textbox", { name: "empty JSON" }), {
    target: { value: '"a JSON string"' },
  });
  expect(JSON.parse(serialized()).empty).toBe("a JSON string");
  expect(screen.getByRole("textbox", { name: "empty JSON" })).toBeTruthy();
});

test("removing an invalid row restores valid JSON without dropping other fields", () => {
  render(<Harness initial='{"retained":false,"finalRules":[]}' />);
  fireEvent.change(screen.getByRole("textbox", { name: "finalRules JSON" }), {
    target: { value: "[" },
  });
  const remove = screen.getAllByRole("button", { name: "删除" })[1];
  if (!remove) throw new Error("Missing invalid row remove button");
  fireEvent.click(remove);
  expect(JSON.parse(serialized())).toEqual({ retained: false });
});

test("external resets replace drafts instead of retaining stale fields", async () => {
  const onChange = vi.fn();
  const view = render(
    <JsonObjectEditor onChange={onChange} value='{"old":[]}' />
  );
  fireEvent.change(screen.getByRole("textbox", { name: "old JSON" }), {
    target: { value: "[" },
  });
  view.rerender(
    <JsonObjectEditor onChange={onChange} value='{"new":{"enabled":true}}' />
  );
  await waitFor(() =>
    expect(screen.queryByRole("textbox", { name: "old JSON" })).toBeNull()
  );
  expect(screen.getByRole("textbox", { name: "new JSON" })).toBeTruthy();
});

test("freedom finalRules survive form extraction and output without extra settings nesting", () => {
  const config = {
    tag: "interdiction-out",
    protocol: "freedom",
    settings: {
      domainStrategy: "AsIs",
      redirect: "127.0.0.1:5080",
      userLevel: 0,
      finalRules: [
        { action: "allow", network: "tcp", ip: ["127.0.0.1/32"], port: "5080" },
      ],
    },
  };
  const values = configToFormValues("outbound", config);
  expect(JSON.parse(String(values.settings_json))).toEqual({
    finalRules: config.settings.finalRules,
  });
  const result = buildOutboundConfig(values);
  expect(result.settings).toEqual(config.settings);
  expect(result.settings).not.toHaveProperty("settings");
});
