import { cn } from "@workspace/ui/lib/utils";
import {
  isDisallowedIntegerKey,
  isIntegerNumberInput,
  normalizeIntegerInputValue,
} from "@workspace/ui/utils/numeric-input";
import type * as React from "react";

function Input({
  className,
  inputMode,
  min,
  onChange,
  onKeyDown,
  step,
  type,
  ...props
}: React.ComponentProps<"input">) {
  const integerInput = isIntegerNumberInput(type, step);
  const allowNegative = min === undefined || Number(min) < 0;

  return (
    <input
      className={cn(
        "h-9 w-full min-w-0 rounded-md border border-input bg-transparent px-3 py-1 text-base shadow-xs outline-none transition-[color,box-shadow] selection:bg-primary selection:text-primary-foreground file:inline-flex file:h-7 file:border-0 file:bg-transparent file:font-medium file:text-foreground file:text-sm placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30",
        "focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50",
        "aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40",
        className
      )}
      data-slot="input"
      inputMode={integerInput ? "numeric" : inputMode}
      min={min}
      onChange={(event) => {
        if (integerInput) {
          const normalized = normalizeIntegerInputValue(
            event.currentTarget.value,
            allowNegative
          );
          if (normalized !== event.currentTarget.value) {
            event.currentTarget.value = normalized;
          }
        }
        onChange?.(event);
      }}
      onKeyDown={(event) => {
        onKeyDown?.(event);
        if (
          integerInput &&
          !event.defaultPrevented &&
          isDisallowedIntegerKey(event.key, allowNegative)
        ) {
          event.preventDefault();
        }
      }}
      step={step}
      type={type}
      {...props}
    />
  );
}

export { Input };
