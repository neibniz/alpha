import { cn } from "@workspace/ui/lib/utils";
import { Loader2Icon } from "lucide-react";
import { useTranslation } from "react-i18next";

function Spinner({ className, ...props }: React.ComponentProps<"svg">) {
  const { t } = useTranslation("components");

  return (
    <Loader2Icon
      aria-label={t("common.loading", "Loading")}
      className={cn("size-4 animate-spin", className)}
      role="status"
      {...props}
    />
  );
}

export { Spinner };
