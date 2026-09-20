import { Button } from "@workspace/ui/components/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { cn } from "@workspace/ui/lib/utils";
import type { LucideIcon } from "lucide-react";
import type { ComponentProps } from "react";

export function AdminActionButton({
  label,
  icon: Icon,
  destructive,
  className,
  ...props
}: ComponentProps<typeof Button> & {
  label: string;
  icon: LucideIcon;
  destructive?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          {...props}
          type="button"
          size="icon-sm"
          variant="ghost"
          aria-label={label}
          className={cn(
            destructive && "text-destructive hover:text-destructive",
            className
          )}
        >
          <Icon className="size-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
