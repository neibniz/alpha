"use client";

import {
  type ColumnFiltersState,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { Checkbox } from "@workspace/ui/components/checkbox";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import Empty from "@workspace/ui/composed/empty";
import {
  ColumnFilter,
  type IParams,
} from "@workspace/ui/composed/pro-list/column-filter";
import { Pagination } from "@workspace/ui/composed/pro-list/pagination";
import { cn } from "@workspace/ui/lib/utils";
import {
  createFilterSnapshot,
  createPageRequest,
  normalizePageResult,
  type PageRequest,
  type PageResult,
} from "@workspace/ui/utils/pagination";
import { AlertCircle, ListRestart, Loader, RefreshCcw } from "lucide-react";
import type React from "react";
import {
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";

export interface ProListProps<TData, TValue> {
  request: (
    pagination: PageRequest,
    filter: TValue
  ) => Promise<PageResult<TData>>;
  params?: IParams[];
  header?: {
    title?: React.ReactNode;
    toolbar?: React.ReactNode | React.ReactNode[];
  };
  batchRender?: (rows: TData[]) => React.ReactNode[];
  renderItem: (item: TData, checkbox: React.ReactNode) => React.ReactNode;
  action?: React.Ref<ProListActions | undefined>;
  texts?: Partial<{
    textRowsPerPage: string;
    textPageOf: (current: number, total: number) => string;
    selectedRowsText: (total: number) => string;
  }>;
  empty?: React.ReactNode;
}
export interface ProListActions {
  refresh: () => void;
  reset: () => void;
}

export function ProList<TData, TValue extends Record<string, unknown>>({
  request,
  params,
  header,
  batchRender,
  renderItem,
  action,
  texts,
  empty,
}: ProListProps<TData, TValue>) {
  const { t } = useTranslation("components");
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [rowSelection, setRowSelection] = useState<{ [key: number]: boolean }>(
    {}
  );
  const [data, setData] = useState<TData[]>([]);
  const [rowCount, setRowCount] = useState<number>(0);
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: 10,
  });
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const requestSequenceRef = useRef(0);
  const requestRef = useRef(request);

  useEffect(() => {
    requestRef.current = request;
  }, [request]);

  const listColumns = useMemo(() => [], []);
  const filterSnapshot = useMemo(
    () => createFilterSnapshot<TValue>(columnFilters),
    [columnFilters]
  );
  const filterValues = filterSnapshot.values;
  const filterSignature = filterSnapshot.signature;

  const table = useReactTable({
    data,
    columns: listColumns,
    onPaginationChange: setPagination,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onRowSelectionChange: setRowSelection,
    state: {
      columnFilters,
      rowSelection,
      pagination,
    },
    manualPagination: true,
    manualFiltering: true,
    rowCount,
  });

  const fetchData = useCallback(async () => {
    const requestSequence = ++requestSequenceRef.current;
    setLoading(true);
    setLoadError(false);
    try {
      const response = await requestRef.current(
        createPageRequest(pagination.pageIndex, pagination.pageSize),
        filterValues as TValue
      );
      const { list, total } = normalizePageResult(response);
      if (requestSequence === requestSequenceRef.current) {
        setData(list);
        setRowCount(total);
        setRowSelection({});
      }
    } catch (error) {
      console.log("Fetch data error:", error);
      if (requestSequence === requestSequenceRef.current) {
        setLoadError(true);
      }
    } finally {
      if (requestSequence === requestSequenceRef.current) {
        setLoading(false);
      }
    }
  }, [filterValues, pagination.pageIndex, pagination.pageSize]);

  const reset = useCallback(async () => {
    setColumnFilters([]);
    setRowSelection({});
    setPagination((previous) => ({ ...previous, pageIndex: 0 }));
  }, []);

  useImperativeHandle(
    action,
    () => ({
      refresh: fetchData,
      reset,
    }),
    [fetchData, reset]
  );

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchData, filterSignature]);

  const handleSelectionChange = useCallback(
    (index: number, isSelected: boolean) => {
      setRowSelection((prevSelection) => ({
        ...prevSelection,
        [index]: isSelected,
      }));
    },
    []
  );

  const selectedRows = useMemo(
    () => data.filter((_, index) => rowSelection[index]),
    [data, rowSelection]
  );
  const selectedCount = selectedRows.length;

  return (
    <div
      className="flex max-w-full flex-col gap-4 overflow-hidden"
      data-slot="pro-list"
    >
      <div className="flex flex-wrap-reverse items-center justify-between gap-4">
        <div className="min-w-0 flex-1">
          {params ? (
            <ColumnFilter
              filters={filterValues}
              params={params}
              table={table}
            />
          ) : (
            header?.title
          )}
        </div>
        <div className="flex items-center justify-end gap-2">
          {params && params?.length > 0 && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    aria-label={t("table.refresh", "Refresh")}
                    onClick={fetchData}
                    size="icon-sm"
                    variant="outline"
                  >
                    <RefreshCcw />
                  </Button>
                </TooltipTrigger>
                <TooltipContent sideOffset={6}>
                  {t("table.refresh", "Refresh")}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    aria-label={t("table.reset", "Reset filters")}
                    onClick={reset}
                    size="icon-sm"
                    variant="outline"
                  >
                    <ListRestart />
                  </Button>
                </TooltipTrigger>
                <TooltipContent sideOffset={6}>
                  {t("table.reset", "Reset filters")}
                </TooltipContent>
              </Tooltip>
            </>
          )}
          {header?.toolbar}
        </div>
      </div>

      {selectedCount > 0 && batchRender && (
        <Alert className="flex items-center justify-between">
          <AlertTitle className="m-0">
            {texts?.selectedRowsText?.(selectedCount) ||
              t("table.selectedRows", "Selected {{count}} rows", {
                count: selectedCount,
              })}
          </AlertTitle>
          <AlertDescription className="flex gap-2">
            {batchRender(selectedRows)}
          </AlertDescription>
        </Alert>
      )}

      {loadError ? (
        <Alert variant="destructive">
          <AlertCircle />
          <AlertTitle>
            {t("table.loadFailed", "Unable to load data")}
          </AlertTitle>
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>
              {t(
                "table.loadFailedDescription",
                "Check the connection and try again."
              )}
            </span>
            <Button onClick={fetchData} size="sm" variant="outline">
              <RefreshCcw />
              {t("table.retry", "Retry")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      <div
        className={cn("relative overflow-x-auto", {
          "rounded-xl border": data.length === 0,
        })}
        data-slot="pro-list-viewport"
      >
        <div className="grid grid-cols-1 gap-4">
          {data.length ? (
            data.map((item, index) => {
              const isSelected = !!rowSelection[index];

              const checkbox = (
                <Checkbox
                  aria-label={t("table.selectRow", "Select row")}
                  checked={isSelected}
                  onCheckedChange={(value) =>
                    handleSelectionChange(index, !!value)
                  }
                />
              );

              return <div key={index}>{renderItem(item, checkbox)}</div>;
            })
          ) : loadError ? null : (
            <div
              className="flex items-center justify-center py-24"
              data-slot="pro-list-empty"
            >
              {empty || <Empty />}
            </div>
          )}
        </div>

        {loading && (
          <div className="absolute top-0 z-20 flex h-full w-full items-center justify-center bg-muted/80">
            <Loader className="h-4 w-4 animate-spin" />
          </div>
        )}
      </div>
      {rowCount > 0 && <Pagination table={table} />}
    </div>
  );
}
