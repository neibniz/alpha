"use client";

import { Button } from "@workspace/ui/components/button";
import { Calendar } from "@workspace/ui/components/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@workspace/ui/components/popover";
import { cn } from "@workspace/ui/lib/utils";
import { intlFormat } from "date-fns";
import { CalendarIcon, X } from "lucide-react";
import * as React from "react";
import type { DayPicker } from "react-day-picker";

export function DatePicker({
  clearLabel = "Clear date",
  placeholder,
  value,
  onChange,
  ...props
}: React.ComponentProps<typeof DayPicker> & {
  placeholder?: string;
  clearLabel?: string;
  value?: number;
  onChange?: (value?: number) => void;
}) {
  const [date, setDate] = React.useState<Date | undefined>(
    value ? new Date(value) : undefined
  );

  React.useEffect(() => {
    setDate(value ? new Date(value) : undefined);
  }, [value]);

  const handleSelect = (selectedDate: Date | undefined) => {
    setDate(selectedDate);
    if (onChange) {
      onChange(selectedDate?.getTime() || 0);
    }
  };

  const handleClear = () => {
    setDate(undefined);
    if (onChange) {
      onChange(0);
    }
  };

  return (
    <div className="relative w-full">
      <Popover>
        <PopoverTrigger asChild>
          <Button
            className={cn(
              "w-full justify-between pr-10 font-normal",
              !value && "text-muted-foreground",
              value && "pr-16"
            )}
            type="button"
            variant="outline"
          >
            {value ? intlFormat(value) : <span>{placeholder}</span>}
            <CalendarIcon aria-hidden="true" className="size-4" />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-0">
          <Calendar
            {...props}
            initialFocus
            mode="single"
            onSelect={handleSelect}
            selected={date}
          />
        </PopoverContent>
      </Popover>
      {value ? (
        <button
          aria-label={clearLabel}
          className="absolute top-1/2 right-9 z-10 inline-flex size-7 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          onClick={handleClear}
          type="button"
        >
          <X aria-hidden="true" className="size-4" />
        </button>
      ) : null}
    </div>
  );
}
