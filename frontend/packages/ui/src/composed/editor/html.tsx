"use client";

import {
  MonacoEditor,
  type MonacoEditorProps,
} from "@workspace/ui/composed/editor/monaco-editor";
import { useTranslation } from "react-i18next";

export function HTMLEditor(props: MonacoEditorProps) {
  const { t } = useTranslation("components");

  return (
    <MonacoEditor
      description={t("editor.html.description", "Support HTML")}
      title={t("editor.html.title", "HTML Editor")}
      {...props}
      language="html"
      render={(value) => (
        <HTMLPreview
          title={t("editor.html.preview", "HTML Preview")}
          value={value}
        />
      )}
    />
  );
}

interface HTMLPreviewProps {
  title?: string;
  value?: string;
}

export function HTMLPreview({
  title = "HTML Preview",
  value,
}: HTMLPreviewProps) {
  return (
    <iframe
      className="h-full w-full border-0"
      sandbox=""
      srcDoc={value || ""}
      title={title}
    />
  );
}
