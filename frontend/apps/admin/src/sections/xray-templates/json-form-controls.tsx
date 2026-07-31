import { Button } from "@workspace/ui/components/button";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Switch } from "@workspace/ui/components/switch";
import { Textarea } from "@workspace/ui/components/textarea";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Braces, TriangleAlert } from "lucide-react";
import { useEffect, useState } from "react";

type JsonObject = Record<string, unknown>;

export type JsonArrayColumn = {
  key: string;
  label: string;
  type?: "text" | "number" | "boolean" | "csv" | "select";
  options?: string[];
  placeholder?: string;
  multiline?: boolean;
  splitMode?: "csv" | "line";
  decodeJsonStringEscapes?: boolean;
  span?: "normal" | "full";
};

type JsonValueType = "string" | "number" | "boolean" | "json";

function formatEditorJson(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function isPlainObject(value: unknown): value is JsonObject {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

export function parseJsonObjectText(value?: string): JsonObject {
  if (!value?.trim()) return {};
  try {
    const parsed = JSON.parse(value);
    return isPlainObject(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

export function parseJsonArrayText<T extends JsonObject = JsonObject>(
  value?: string
): T[] {
  if (!value?.trim()) return [];
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? (parsed.filter(isPlainObject) as T[]) : [];
  } catch {
    return [];
  }
}

function hasInvalidJson(
  value: string | undefined,
  expected: "array" | "object"
) {
  if (!value?.trim()) return false;
  try {
    const parsed = JSON.parse(value);
    return expected === "array"
      ? !Array.isArray(parsed)
      : !isPlainObject(parsed);
  } catch {
    return true;
  }
}

function decodeJsonStringEscapes(value: string) {
  if (!value.includes("\\")) return value;
  try {
    const decoded = JSON.parse(`"${value.replace(/"/g, '\\"')}"`);
    return typeof decoded === "string" ? decoded : value;
  } catch {
    return value;
  }
}

function splitListText(value: string, splitMode: JsonArrayColumn["splitMode"]) {
  return splitMode === "line" ? value.split(/\r?\n+/) : value.split(/[\n,]+/);
}

function normalizeListItem(value: string, column?: JsonArrayColumn) {
  const trimmed = value.trim();
  if (!trimmed) return "";
  return column?.decodeJsonStringEscapes
    ? decodeJsonStringEscapes(trimmed)
    : trimmed;
}

function csvToArray(value: string, column?: JsonArrayColumn) {
  return splitListText(value, column?.splitMode)
    .map((item) => normalizeListItem(item, column))
    .filter(Boolean);
}

function arrayToCsv(value: unknown) {
  return Array.isArray(value) ? value.join(", ") : String(value ?? "");
}

function arrayToLines(value: unknown) {
  return Array.isArray(value) ? value.join("\n") : String(value ?? "");
}

function columnClassName(column: JsonArrayColumn) {
  return column.span === "full" || column.multiline
    ? "space-y-1 md:col-span-2"
    : "space-y-1";
}

function MultilineCsvTextarea({
  column,
  onValueChange,
  placeholder,
  value,
}: {
  column: JsonArrayColumn;
  onValueChange: (value: string[]) => void;
  placeholder?: string;
  value: unknown;
}) {
  const externalValue = arrayToLines(value);
  const [draft, setDraft] = useState(externalValue);
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    if (!focused) {
      setDraft(externalValue);
    }
  }, [externalValue, focused]);

  function commit(text = draft) {
    onValueChange(csvToArray(text, column));
  }

  return (
    <div className="space-y-1">
      <Textarea
        className="min-h-32 resize-y whitespace-pre-wrap font-mono text-xs leading-5"
        onBlur={() => {
          setFocused(false);
          commit();
        }}
        onChange={(event) => {
          const next = event.target.value;
          setDraft(next);
          commit(next);
        }}
        onFocus={() => setFocused(true)}
        placeholder={placeholder}
        value={draft}
      />
      <div className="text-[11px] text-muted-foreground">
        {column.splitMode === "line"
          ? "每行一条。"
          : "每行一条，也支持逗号分隔。"}
      </div>
    </div>
  );
}

function inferValueType(value: unknown): JsonValueType {
  if (typeof value === "number") return "number";
  if (typeof value === "boolean") return "boolean";
  if (value && typeof value === "object") return "json";
  return "string";
}

function valueToText(value: unknown, type: JsonValueType) {
  if (value === undefined || value === null) return "";
  if (type === "json") return formatEditorJson(value);
  return String(value);
}

function parseTypedValue(text: string, type: JsonValueType) {
  if (type === "number") {
    const value = Number(text);
    return Number.isNaN(value) ? undefined : value;
  }
  if (type === "boolean") return text === "true";
  if (type === "json") {
    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }
  return text;
}

function nextUniqueKey(source: JsonObject, prefix = "key") {
  let index = Object.keys(source).length + 1;
  let key = `${prefix}_${index}`;
  while (key in source) {
    index += 1;
    key = `${prefix}_${index}`;
  }
  return key;
}

export function JsonObjectEditor({
  addLabel = "添加字段",
  emptyText = "暂无字段。",
  onChange,
  value,
  valuePlaceholder = "值",
}: {
  addLabel?: string;
  emptyText?: string;
  onChange: (value: string) => void;
  value?: string;
  valuePlaceholder?: string;
}) {
  const objectValue = parseJsonObjectText(value);
  const entries = Object.entries(objectValue);

  function commit(next: JsonObject) {
    onChange(formatEditorJson(next));
  }

  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-3">
      {hasInvalidJson(value, "object") ? (
        <Alert>
          <TriangleAlert />
          <AlertDescription>
            当前内容不是合法 JSON 对象，表单暂时按空对象处理。
          </AlertDescription>
        </Alert>
      ) : null}
      {entries.length ? (
        entries.map(([key, currentValue]) => {
          const type = inferValueType(currentValue);
          return (
            <div
              className="space-y-2 rounded-md border bg-background p-2"
              key={key}
            >
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_120px]">
                <EnhancedInput
                  onValueChange={(nextKey) => {
                    const normalized = nextKey.trim();
                    if (!normalized || normalized === key) return;
                    const next = { ...objectValue };
                    delete next[key];
                    next[normalized] = currentValue;
                    commit(next);
                  }}
                  placeholder="字段名"
                  value={key}
                />
                <Select
                  onValueChange={(nextType) => {
                    const next = {
                      ...objectValue,
                      [key]: parseTypedValue(
                        valueToText(currentValue, type),
                        nextType as JsonValueType
                      ),
                    };
                    commit(next);
                  }}
                  value={type}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="string">文本</SelectItem>
                    <SelectItem value="number">数字</SelectItem>
                    <SelectItem value="boolean">开关</SelectItem>
                    <SelectItem value="json">JSON</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
                {type === "boolean" ? (
                  <div className="flex h-9 items-center rounded-md border px-3">
                    <Switch
                      checked={!!currentValue}
                      onCheckedChange={(checked) =>
                        commit({ ...objectValue, [key]: checked })
                      }
                    />
                  </div>
                ) : type === "json" ? (
                  <Textarea
                    className="min-h-20 font-mono text-xs"
                    onChange={(event) =>
                      commit({
                        ...objectValue,
                        [key]: parseTypedValue(event.target.value, "json"),
                      })
                    }
                    placeholder='{"enabled":true}'
                    value={valueToText(currentValue, type)}
                  />
                ) : (
                  <EnhancedInput
                    onValueChange={(text) =>
                      commit({
                        ...objectValue,
                        [key]: parseTypedValue(text, type),
                      })
                    }
                    placeholder={valuePlaceholder}
                    type={type === "number" ? "number" : "text"}
                    value={valueToText(currentValue, type)}
                  />
                )}
                <Button
                  className="h-9 sm:px-3"
                  onClick={() => {
                    const next = { ...objectValue };
                    delete next[key];
                    commit(next);
                  }}
                  type="button"
                  variant="outline"
                >
                  删除
                </Button>
              </div>
            </div>
          );
        })
      ) : (
        <Empty className="border p-4">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Braces />
            </EmptyMedia>
            <EmptyDescription>{emptyText}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      <Button
        onClick={() =>
          commit({ ...objectValue, [nextUniqueKey(objectValue)]: "" })
        }
        size="sm"
        type="button"
        variant="outline"
      >
        {addLabel}
      </Button>
    </div>
  );
}

export function JsonArrayObjectEditor({
  addLabel = "添加一项",
  columns,
  defaultItem = {},
  emptyText = "暂无项目。",
  onChange,
  value,
}: {
  addLabel?: string;
  columns: JsonArrayColumn[];
  defaultItem?: JsonObject;
  emptyText?: string;
  onChange: (value: string) => void;
  value?: string;
}) {
  const items = parseJsonArrayText(value);

  function commit(next: JsonObject[]) {
    onChange(formatEditorJson(next));
  }

  function updateItem(index: number, key: string, nextValue: unknown) {
    const next = items.map((item, itemIndex) => {
      if (itemIndex !== index) return item;
      const updated = { ...item };
      if (
        nextValue === "" ||
        nextValue === undefined ||
        (Array.isArray(nextValue) && nextValue.length === 0)
      ) {
        delete updated[key];
      } else {
        updated[key] = nextValue;
      }
      return updated;
    });
    commit(next);
  }

  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-3">
      {hasInvalidJson(value, "array") ? (
        <Alert>
          <TriangleAlert />
          <AlertDescription>
            当前内容不是合法 JSON 数组，表单暂时按空数组处理。
          </AlertDescription>
        </Alert>
      ) : null}
      {items.length ? (
        items.map((item, index) => (
          <div
            className="space-y-2 rounded-md border bg-background p-3"
            key={index}
          >
            <div className="flex items-center justify-between gap-2">
              <div className="font-medium text-sm">#{index + 1}</div>
              <Button
                onClick={() =>
                  commit(items.filter((_, itemIndex) => itemIndex !== index))
                }
                size="sm"
                type="button"
                variant="outline"
              >
                删除
              </Button>
            </div>
            <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
              {columns.map((column) => {
                const rawValue = item[column.key];
                if (column.type === "boolean") {
                  return (
                    <div
                      className={`flex items-center justify-between rounded-md border px-3 py-2 ${
                        column.span === "full" ? "md:col-span-2" : ""
                      }`}
                      key={column.key}
                    >
                      <span className="text-sm">{column.label}</span>
                      <Switch
                        checked={!!rawValue}
                        onCheckedChange={(checked) =>
                          updateItem(index, column.key, checked)
                        }
                      />
                    </div>
                  );
                }
                if (column.type === "select") {
                  return (
                    <div className={columnClassName(column)} key={column.key}>
                      <div className="text-muted-foreground text-xs">
                        {column.label}
                      </div>
                      <Select
                        onValueChange={(nextValue) =>
                          updateItem(index, column.key, nextValue)
                        }
                        value={String(rawValue ?? "")}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {(column.options || []).map((option) => (
                            <SelectItem key={option} value={option}>
                              {option}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  );
                }
                return (
                  <div className={columnClassName(column)} key={column.key}>
                    <div className="text-muted-foreground text-xs">
                      {column.label}
                    </div>
                    {column.multiline ? (
                      <MultilineCsvTextarea
                        column={column}
                        onValueChange={(nextValue) =>
                          updateItem(index, column.key, nextValue)
                        }
                        placeholder={column.placeholder}
                        value={rawValue}
                      />
                    ) : (
                      <EnhancedInput
                        onValueChange={(text) => {
                          if (column.type === "number") {
                            const numberValue = Number(text);
                            updateItem(
                              index,
                              column.key,
                              Number.isNaN(numberValue)
                                ? undefined
                                : numberValue
                            );
                            return;
                          }
                          updateItem(
                            index,
                            column.key,
                            column.type === "csv"
                              ? csvToArray(text, column)
                              : text
                          );
                        }}
                        placeholder={column.placeholder}
                        type={column.type === "number" ? "number" : "text"}
                        value={
                          column.type === "csv"
                            ? arrayToCsv(rawValue)
                            : String(rawValue ?? "")
                        }
                      />
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        ))
      ) : (
        <Empty className="border p-4">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Braces />
            </EmptyMedia>
            <EmptyDescription>{emptyText}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      <Button
        onClick={() => commit([...items, { ...defaultItem }])}
        size="sm"
        type="button"
        variant="outline"
      >
        {addLabel}
      </Button>
    </div>
  );
}
