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
import { Input } from "@workspace/ui/components/input";
import { Braces, TriangleAlert } from "lucide-react";
import { useCallback, useEffect, useId, useRef, useState } from "react";

type JsonObject = Record<string, unknown>;

export type JsonArrayColumn = {
  key: string;
  label: string;
  type?: "text" | "number" | "integer" | "boolean" | "csv" | "select";
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
  if (value === null || (value && typeof value === "object")) return "json";
  return "string";
}

function valueToText(value: unknown, type: JsonValueType) {
  if (value === undefined || value === null) return "";
  if (type === "json") return formatEditorJson(value);
  return String(value);
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

type JsonFieldDraft = {
  id: number;
  key: string;
  type: JsonValueType;
  text: string;
};

function draftError(row: JsonFieldDraft, rows: JsonFieldDraft[]) {
  if (!row.key.trim()) return "字段名不能为空。";
  if (rows.some((other) => other.id !== row.id && other.key === row.key)) {
    return "字段名不能重复。";
  }
  if (row.type === "json") {
    try {
      JSON.parse(row.text);
    } catch {
      return "请输入完整、有效的 JSON。";
    }
  }
  if (
    row.type === "number" &&
    !(row.text.trim() && Number.isFinite(Number(row.text)))
  ) {
    return "请输入有效的有限数字。";
  }
  return undefined;
}

function serializeDrafts(rows: JsonFieldDraft[]) {
  // Invalid drafts must reach form validation, not silently become valid strings.
  if (rows.some((row) => draftError(row, rows))) return "{";
  return `{\n${rows
    .map((row) => {
      const text =
        row.type === "string"
          ? JSON.stringify(row.text)
          : row.type === "number"
            ? JSON.stringify(Number(row.text))
            : row.text;
      return `  ${JSON.stringify(row.key)}: ${text}`;
    })
    .join(",\n")}\n}`;
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
  const nextId = useRef(0);
  const editorId = useId();
  const emittedValue = useRef(value);
  const externalValue = useRef(value);
  const readRows = useCallback((source?: string): JsonFieldDraft[] => {
    return Object.entries(parseJsonObjectText(source)).map(([key, current]) => {
      const type = inferValueType(current);
      return {
        id: nextId.current++,
        key,
        type,
        text:
          type === "json" && current === null
            ? "null"
            : valueToText(current, type),
      };
    });
  }, []);
  const [rows, setRows] = useState(() => readRows(value));

  useEffect(() => {
    if (value === externalValue.current) return;
    externalValue.current = value;
    if (value !== emittedValue.current) setRows(readRows(value));
  }, [readRows, value]);

  function commit(next: JsonFieldDraft[]) {
    setRows(next);
    emittedValue.current = serializeDrafts(next);
    onChange(emittedValue.current);
  }

  function update(id: number, patch: Partial<JsonFieldDraft>) {
    commit(rows.map((row) => (row.id === id ? { ...row, ...patch } : row)));
  }

  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-3">
      {hasInvalidJson(value, "object") ? (
        <Alert>
          <TriangleAlert />
          <AlertDescription>
            存在未完成或无效的 JSON，请修正后保存。
          </AlertDescription>
        </Alert>
      ) : null}
      {rows.length ? (
        rows.map((row) => {
          const { id, key, type, text } = row;
          const error = draftError(row, rows);
          const errorId = `${editorId}-${id}-error`;
          return (
            <div
              className="space-y-2 rounded-md border bg-background p-2"
              key={id}
            >
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_120px]">
                <Input
                  aria-label="字段名"
                  aria-describedby={error ? errorId : undefined}
                  aria-invalid={!!error}
                  onChange={(event) => update(id, { key: event.target.value })}
                  placeholder="字段名"
                  value={key}
                />
                <Select
                  onValueChange={(nextType) => {
                    const selected = nextType as JsonValueType;
                    let nextText = text;
                    if (selected === "json" && !text.trim()) nextText = "{}";
                    if (selected === "number") {
                      nextText = Number.isFinite(Number(text))
                        ? String(Number(text))
                        : "0";
                    }
                    if (selected === "boolean")
                      nextText = String(text === "true");
                    update(id, { type: selected, text: nextText });
                  }}
                  value={type}
                >
                  <SelectTrigger aria-label="字段类型">
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
                      aria-label={`${key} 开关`}
                      checked={text === "true"}
                      onCheckedChange={(checked) =>
                        update(id, { text: String(checked) })
                      }
                    />
                  </div>
                ) : type === "json" ? (
                  <Textarea
                    aria-label={`${key} JSON`}
                    aria-describedby={error ? errorId : undefined}
                    aria-invalid={!!error}
                    className="min-h-20 font-mono text-xs"
                    onChange={(event) =>
                      update(id, { text: event.target.value })
                    }
                    placeholder='{"enabled":true}'
                    value={text}
                  />
                ) : (
                  <Input
                    aria-label={`${key} 值`}
                    aria-describedby={error ? errorId : undefined}
                    aria-invalid={!!error}
                    inputMode={type === "number" ? "decimal" : undefined}
                    onChange={(event) =>
                      update(id, { text: event.target.value })
                    }
                    placeholder={valuePlaceholder}
                    value={text}
                  />
                )}
                <Button
                  className="h-9 sm:px-3"
                  onClick={() => {
                    commit(rows.filter((current) => current.id !== id));
                  }}
                  type="button"
                  variant="outline"
                >
                  删除
                </Button>
              </div>
              {error ? (
                <p
                  className="text-sm text-destructive"
                  id={errorId}
                  role="alert"
                >
                  {error}
                </p>
              ) : null}
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
        onClick={() => {
          const keys = Object.fromEntries(rows.map((row) => [row.key, true]));
          commit([
            ...rows,
            {
              id: nextId.current++,
              key: nextUniqueKey(keys),
              type: "string",
              text: "",
            },
          ]);
        }}
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
                        min={column.type === "integer" ? 0 : undefined}
                        onValueChange={(text) => {
                          if (
                            column.type === "number" ||
                            column.type === "integer"
                          ) {
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
                        step={column.type === "integer" ? 1 : undefined}
                        type={
                          column.type === "number" || column.type === "integer"
                            ? "number"
                            : "text"
                        }
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
