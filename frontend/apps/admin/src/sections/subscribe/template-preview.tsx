"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Icon } from "@workspace/ui/composed/icon";
import { previewSubscribeTemplate } from "@workspace/ui/services/admin/application";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LazyMonacoEditor } from "@/components/lazy-monaco-editor";

interface TemplatePreviewProps {
  applicationId: number;
  output_format?: string;
  compact?: boolean;
}

function getQueryErrorMessage(error: unknown) {
  if (!error || typeof error !== "object") return;

  const data = "data" in error ? error.data : undefined;
  if (data && typeof data === "object" && "msg" in data) {
    const message = data.msg;
    if (typeof message === "string") return message;
  }

  if ("message" in error && typeof error.message === "string") {
    return error.message;
  }

  return;
}

export function TemplatePreview({
  applicationId,
  output_format,
  compact = false,
}: TemplatePreviewProps) {
  const { t } = useTranslation("subscribe");
  const [isOpen, setIsOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ["previewSubscribeTemplate", applicationId],
    queryFn: () =>
      previewSubscribeTemplate(
        { id: applicationId },
        { skipErrorHandler: true }
      ),
    enabled: isOpen && !!applicationId,
    retry: false,
  });

  const originalContent = data?.data?.data?.template || "";
  const errorMessage =
    getQueryErrorMessage(error) ||
    t("templatePreview.failed", "Failed to load template");

  const getDecodedContent = () => {
    if (output_format === "base64" && originalContent) {
      try {
        return atob(originalContent);
      } catch {
        return t("templatePreview.base64.decodeError", "Base64 decode error");
      }
    }
    return "";
  };

  const getDisplayContent = () => {
    if (error) return errorMessage;
    if (!originalContent) return "";
    switch (output_format) {
      case "base64": {
        const decoded = getDecodedContent();
        return `${t("templatePreview.base64.originalContent", "Original Content")}:\n${originalContent}\n\n${t("templatePreview.base64.decodedContent", "Decoded Content")}:\n${decoded}`;
      }
      default:
        return originalContent;
    }
  };
  const mapLanguage = (fmt?: string) => {
    switch (fmt) {
      case "json":
        return "json";
      case "yaml":
        return "yaml";
      case "base64":
        return "ini";
      case "plain":
        return "ini";
      case "conf":
        return "ini";
      default:
        return "ini";
    }
  };

  return (
    <Sheet onOpenChange={setIsOpen} open={isOpen}>
      <SheetTrigger asChild>
        <Button
          aria-label={t("templatePreview.preview", "Preview")}
          size={compact ? "icon-sm" : "default"}
          title={t("templatePreview.preview", "Preview")}
          variant="ghost"
        >
          <Icon className="h-4 w-4" icon="mdi:eye" />
          {compact ? null : t("templatePreview.preview", "Preview")}
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[800px] max-w-[90vw] md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>
            {t("templatePreview.title", "Template Preview")}
          </SheetTitle>
          <SheetDescription>
            {t(
              "templatePreview.description",
              "Preview the rendered output and verify its structure before publishing"
            )}
          </SheetDescription>
        </SheetHeader>
        {isLoading ? (
          <div className="flex items-center justify-center">
            <Icon className="h-6 w-6 animate-spin" icon="mdi:loading" />
            <span className="ml-2">
              {t("templatePreview.loading", "Loading...")}
            </span>
          </div>
        ) : (
          <LazyMonacoEditor
            language={mapLanguage(output_format)}
            readOnly
            showLineNumbers
            title={t("templatePreview.title", "Template Preview")}
            value={getDisplayContent()}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
