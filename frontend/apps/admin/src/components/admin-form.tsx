import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { cn } from "@workspace/ui/lib/utils";
import { CircleAlert } from "lucide-react";
import type { ComponentProps } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { useTranslation } from "react-i18next";

function collectMessages(value: unknown): string[] {
  if (!value || typeof value !== "object") return [];
  if ("message" in value && typeof value.message === "string")
    return [value.message];
  return Object.entries(value)
    .filter(([key]) => !["ref", "types"].includes(key))
    .flatMap(([, nested]) => collectMessages(nested));
}

export function AdminForm({
  children,
  className,
  ...props
}: ComponentProps<"form">) {
  const { control } = useFormContext();
  const { errors, isSubmitting, submitCount } = useFormState({ control });
  const { t } = useTranslation("components");
  const messages = submitCount > 0 ? [...new Set(collectMessages(errors))] : [];
  return (
    <form
      {...props}
      aria-busy={isSubmitting}
      noValidate
      className={cn("admin-form flex flex-col gap-5", className)}
    >
      {messages.length > 0 && (
        <Alert className="col-span-full" variant="destructive">
          <CircleAlert />
          <AlertTitle>
            {t("form.reviewErrors", "Review the highlighted fields")}
          </AlertTitle>
          <AlertDescription>
            <ul className="list-inside list-disc">
              {messages.map((message) => (
                <li key={message}>{message}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}
      {children}
    </form>
  );
}
