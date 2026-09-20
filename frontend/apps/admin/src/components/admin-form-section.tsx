import {
  FieldGroup,
  FieldLegend,
  FieldSet,
} from "@workspace/ui/components/field";
import type { ReactNode } from "react";

export function AdminFormSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <FieldSet className="min-w-0 border-b pb-6 last:border-0">
      <FieldLegend className="mb-0">{title}</FieldLegend>
      {description && (
        <p className="text-sm leading-6 text-muted-foreground">{description}</p>
      )}
      <FieldGroup>{children}</FieldGroup>
    </FieldSet>
  );
}
