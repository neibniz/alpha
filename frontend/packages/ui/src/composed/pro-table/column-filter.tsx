"use client";

import type { Table } from "@tanstack/react-table";
import { Input } from "@workspace/ui/components/input";
import { Combobox } from "@workspace/ui/composed/combobox";
import { cn } from "@workspace/ui/lib/utils";
import { useTranslation } from "react-i18next";

export interface IParams {
  key: string;
  placeholder?: string;
  options?: { label: string; value: string }[];
  type?: "text" | "select" | "date";
  className?: string;
}
interface ColumnFilterProps<TData> {
  table: Table<TData>;
  params: IParams[];
  filters?: Record<string, unknown>;
}

export function ColumnFilter<TData>({
  table,
  params,
  filters = {},
}: ColumnFilterProps<TData>) {
  const { t } = useTranslation("components");

  const updateFilter = (key: string, value: unknown) => {
    table.setColumnFilters((prev) => {
      const newFilters = prev.filter((filter) => filter.id !== key);
      if (value) {
        newFilters.push({ id: key, value });
      }
      return newFilters;
    });
  };

  const toDateInput = (d: Date) => {
    const pad = (n: number) => String(n).padStart(2, "0");
    const yyyy = d.getFullYear();
    const MM = pad(d.getMonth() + 1);
    const dd = pad(d.getDate());
    return `${yyyy}-${MM}-${dd}`;
  };

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2">
      {params.map((param) => {
        const filterValue = filters[param.key];
        if (param.options || param.type === "select") {
          return (
            <Combobox
              className={cn("w-full min-w-36 sm:w-44", param.className)}
              key={param.key}
              onChange={(value) => {
                updateFilter(param.key, value);
              }}
              options={param.options}
              placeholder={param.placeholder || t("filter.choose", "Choose...")}
              value={typeof filterValue === "string" ? filterValue : ""}
            />
          );
        }
        if (param.type === "date") {
          const raw = filterValue;
          const inputValue =
            typeof raw === "number"
              ? toDateInput(new Date(raw))
              : typeof raw === "string"
                ? raw
                : "";
          return (
            <Input
              className={cn("block w-full min-w-36 sm:w-44", param.className)}
              key={param.key}
              onChange={(event) => {
                const v = event.target.value;
                updateFilter(param.key, v || "");
              }}
              placeholder={param.placeholder}
              type="date"
              value={inputValue}
            />
          );
        }
        return (
          <Input
            className={cn("w-full min-w-36 sm:w-44", param.className)}
            key={param.key}
            onChange={(event) => updateFilter(param.key, event.target.value)}
            placeholder={param.placeholder || t("filter.search", "Search...")}
            value={
              typeof filterValue === "string" || typeof filterValue === "number"
                ? filterValue
                : ""
            }
          />
        );
      })}
    </div>
  );
}
