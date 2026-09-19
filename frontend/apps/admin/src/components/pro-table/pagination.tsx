import type { Table } from "@tanstack/react-table";
import { Button } from "@workspace/ui/components/button";
import { Input } from "@workspace/ui/components/input";
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
import { useEffect, useState } from "react";

interface PaginationProps<TData> {
  table: Table<TData>;
}

export function Pagination<TData>({ table }: PaginationProps<TData>) {
  const { t } = useTranslation("components");
  const currentPage = table.getState().pagination.pageIndex + 1;
  const pageCount = Math.max(1, table.getPageCount());
  const [page, setPage] = useState(String(currentPage));
  useEffect(() => setPage(String(currentPage)), [currentPage]);
  const goToPage = () => {
    const requested = Number(page);
    if (
      !Number.isSafeInteger(requested) ||
      requested < 1 ||
      requested > pageCount
    ) {
      setPage(String(currentPage));
      return;
    }
    table.setPageIndex(requested - 1);
  };

  return (
    <div
      className="flex flex-wrap items-center justify-between gap-3 text-sm"
      data-slot="pro-table-pagination"
    >
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
            <SelectTrigger
              aria-label={t("pagination.rowsPerPage", "Rows per page")}
              className="w-20"
              size="sm"
            >
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
        <Input
          aria-label={t("pagination.selectPageNumber", "Select page number")}
          className="h-8 w-16 text-center tabular-nums"
          max={pageCount}
          min={1}
          onBlur={goToPage}
          onChange={(event) => setPage(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              goToPage();
            }
          }}
          step={1}
          type="number"
          value={page}
        />
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
