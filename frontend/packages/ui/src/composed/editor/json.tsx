"use client";

import {
  MonacoEditor,
  type MonacoEditorProps,
} from "@workspace/ui/composed/editor/monaco-editor";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

interface JSONEditorProps
  extends Omit<MonacoEditorProps, "placeholder" | "value" | "onChange"> {
  schema?: Record<string, unknown>;
  placeholder?: Record<string, unknown>;
  value?: Record<string, unknown> | string;
  onChange?: (value: Record<string, unknown> | string | undefined) => void;
}

export function JSONEditor(props: JSONEditorProps) {
  const { t } = useTranslation("components");
  const { schema, placeholder = {}, ...rest } = props;

  const editorKey = useMemo(
    () => JSON.stringify({ schema, placeholder }),
    [schema, placeholder]
  );

  const handleMount: MonacoEditorProps["onMount"] = (editor, monaco) => {
    props.onMount?.(editor, monaco);
    const jsonDefaults = monaco.languages.json?.jsonDefaults;
    if (!jsonDefaults) {
      console.warn(
        "Monaco JSON diagnostics are unavailable; continuing without schema validation."
      );
      return;
    }
    jsonDefaults.setDiagnosticsOptions({
      validate: true,
      schemas: [
        {
          uri: "",
          fileMatch: ["*"],
          schema: schema || {
            type: "object",
            properties: generateSchema(placeholder),
          },
        },
      ],
    });
  };

  return (
    <MonacoEditor
      key={editorKey}
      title={t("editor.json.title", "Edit JSON")}
      {...rest}
      language="json"
      onChange={(value) => {
        if (props.onChange && typeof value === "string") {
          try {
            props.onChange(
              typeof props.value === "string" ? value : JSON.parse(value)
            );
          } catch (e) {
            console.log("Invalid JSON input:", e);
          }
        }
      }}
      onMount={handleMount}
      placeholder={placeholder ? JSON.stringify(placeholder, null, 2) : ""}
      value={
        typeof props.value === "string"
          ? props.value
          : props.value
            ? JSON.stringify(props.value, null, 2)
            : ""
      }
    />
  );
}

const generateSchema = (
  obj: Record<string, unknown>
): Record<string, SchemaProperty> => {
  const properties: Record<string, SchemaProperty> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (Array.isArray(value)) {
      properties[key] = {
        type: "array",
        items:
          value.length > 0
            ? generateSchema({ item: value[0] }).item
            : { type: "null" },
      };
    } else if (typeof value === "object" && value !== null) {
      properties[key] = {
        type: "object",
        properties: generateSchema(value as Record<string, unknown>),
      };
    } else {
      properties[key] = { type: typeof value as SchemaType };
    }
  }
  return properties;
};

type SchemaType = "string" | "number" | "boolean" | "object" | "array" | "null";
interface SchemaProperty {
  type: SchemaType;
  items?: SchemaProperty;
  properties?: Record<string, SchemaProperty>;
}
