import type { Table } from "@tanstack/react-table";
import { Button } from "@workspace/ui/components/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";

interface PaginationProps<TData> {
  table: Table<TData>;
}

export function Pagination<TData>({ table }: PaginationProps<TData>) {
  const { t } = useTranslation("components");

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
      <div className="min-w-32 flex-1 whitespace-nowrap text-muted-foreground">
        {t("pagination.pageInfo", "Page {{page}} of {{total}}", {
          page: table.getState().pagination.pageIndex + 1,
          total: table.getPageCount(),
        })}
      </div>
      <div className="flex flex-grow flex-wrap items-center justify-end gap-1.5">
        <div className="mr-1 flex items-center gap-2">
          <p className="hidden font-medium text-muted-foreground md:block">
            {t("pagination.rowsPerPage", "Rows per page")}
          </p>
          <Select
            onValueChange={(value) => {
              table.setPageSize(Number(value));
            }}
            value={`${table.getState().pagination.pageSize}`}
          >
            <SelectTrigger className="w-16" size="sm">
              <SelectValue placeholder={table.getState().pagination.pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectGroup>
                {[10, 20, 50, 100, 200].map((pageSize) => (
                  <SelectItem key={pageSize} value={`${pageSize}`}>
                    {pageSize}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <Button
          className="hidden lg:flex"
          disabled={!table.getCanPreviousPage()}
          onClick={() => table.setPageIndex(0)}
          size="icon-sm"
          variant="outline"
        >
          <span className="sr-only">
            {t("pagination.firstPage", "Go to first page")}
          </span>
          <ChevronsLeftIcon />
        </Button>
        <Button
          disabled={!table.getCanPreviousPage()}
          onClick={() => table.previousPage()}
          size="icon-sm"
          variant="outline"
        >
          <span className="sr-only">
            {t("pagination.previousPage", "Go to previous page")}
          </span>
          <ChevronLeftIcon />
        </Button>
        <Select
          onValueChange={(value) => table.setPageIndex(Number(value) - 1)}
          value={`${table.getState().pagination.pageIndex + 1}`}
        >
          <SelectTrigger className="w-16" size="sm">
            <SelectValue
              placeholder={t(
                "pagination.selectPageNumber",
                "Select page number"
              )}
            />
          </SelectTrigger>
          <SelectContent className="w-16">
            <SelectGroup>
              {Array.from({ length: table.getPageCount() }, (_, i) => (
                <SelectItem key={i} value={`${i + 1}`}>
                  {i + 1}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Button
          disabled={!table.getCanNextPage()}
          onClick={() => table.nextPage()}
          size="icon-sm"
          variant="outline"
        >
          <span className="sr-only">
            {t("pagination.nextPage", "Go to next page")}
          </span>
          <ChevronRightIcon />
        </Button>
        <Button
          className="hidden lg:flex"
          disabled={!table.getCanNextPage()}
          onClick={() => table.setPageIndex(table.getPageCount() - 1)}
          size="icon-sm"
          variant="outline"
        >
          <span className="sr-only">
            {t("pagination.lastPage", "Go to last page")}
          </span>
          <ChevronsRightIcon />
        </Button>
      </div>
    </div>
  );
}
