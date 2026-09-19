"use client";

import { Button } from "@workspace/ui/components/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@workspace/ui/components/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@workspace/ui/components/popover";
import { cn } from "@workspace/ui/lib/utils";
import { BoxIcon, CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";

export type Option<T = string> = {
  value: T;
  label: string;
  children?: React.ReactNode;
};

// Conditional types to determine the value type for onChange
type OnChangeType<T, M extends boolean> = M extends true ? T[] : T;

type ComboboxProps<T = string, M extends boolean = false> = {
  "aria-label"?: string;
  multiple?: M;
  options?: Option<T>[];
  placeholder?: string;
  value?: OnChangeType<T, M>;
  onChange: (value: OnChangeType<T, M>) => void;
  className?: string;
};

export function Combobox<T, M extends boolean = false>({
  multiple = false as M,
  options = [],
  placeholder,
  value,
  onChange,
  className,
  "aria-label": ariaLabel,
}: ComboboxProps<T, M>) {
  const { t } = useTranslation("components");
  const [open, setOpen] = React.useState(false);
  const resolvedPlaceholder = placeholder ?? t("combobox.select", "Select...");
  const optionByValue = React.useMemo(() => {
    const result = new Map<T, Option<T>>();
    for (const option of options) {
      result.set(option.value, option);
    }
    return result;
  }, [options]);
  const selectedValueSet = React.useMemo(() => {
    if (!(multiple && Array.isArray(value))) {
      return;
    }
    return new Set<T>(value);
  }, [multiple, value]);

  const handleSelect = (selectedValue: T) => {
    if (multiple) {
      const newValue = Array.isArray(value) ? [...value] : [];

      if (selectedValueSet?.has(selectedValue)) {
        onChange(
          newValue.filter((item) => item !== selectedValue) as OnChangeType<
            T,
            M
          >
        );
      } else {
        onChange([...newValue, selectedValue] as OnChangeType<T, M>);
      }
    } else {
      const newValue = selectedValue === value ? ("" as T) : selectedValue;
      onChange(newValue as OnChangeType<T, M>);
      setOpen(false);
    }
  };

  const renderButtonLabel = () => {
    if (multiple && selectedValueSet && selectedValueSet.size > 0) {
      const selectedLabels: string[] = [];
      for (const option of options) {
        if (selectedValueSet.has(option.value)) {
          selectedLabels.push(option.label);
        }
      }

      return selectedLabels.join(", ");
    }
    if (!multiple) {
      const selectedOption = optionByValue.get(value as T);

      return selectedOption
        ? selectedOption.children || selectedOption.label
        : resolvedPlaceholder;
    }

    return resolvedPlaceholder;
  };

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <PopoverTrigger asChild>
        <Button
          aria-expanded={open}
          aria-label={ariaLabel || resolvedPlaceholder}
          className={cn("w-full items-center justify-between", className)}
          role="combobox"
          variant="outline"
        >
          <span className="truncate">{renderButtonLabel()}</span>
          <ChevronsUpDownIcon className="ml-2 size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-fit p-0">
        <Command>
          <CommandInput
            className="h-9"
            placeholder={t("combobox.search", "Search...")}
          />
          <CommandEmpty>
            <BoxIcon className="inline-block text-slate-500" />
          </CommandEmpty>
          <CommandGroup>
            <CommandList
              onWheel={(e) => {
                e.currentTarget.scrollTop += e.deltaY;
              }}
            >
              {options.map((option) => (
                <CommandItem
                  key={String(option.label + option.value)}
                  onSelect={() => handleSelect(option.value)}
                  value={option.label + option.value}
                >
                  {option.children || option.label}
                  <CheckIcon
                    className={cn(
                      "ml-auto h-4 w-4",
                      multiple
                        ? selectedValueSet?.has(option.value)
                          ? "opacity-100"
                          : "opacity-0"
                        : value === option.value
                          ? "opacity-100"
                          : "opacity-0"
                    )}
                  />
                </CommandItem>
              ))}
            </CommandList>
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
