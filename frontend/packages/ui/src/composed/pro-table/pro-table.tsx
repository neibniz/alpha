"use client";

import {
  type CellContext,
  type ColumnDef,
  type ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Checkbox } from "@workspace/ui/components/checkbox";
import { Separator } from "@workspace/ui/components/separator";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@workspace/ui/components/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import Empty from "@workspace/ui/composed/empty";
import {
  ColumnFilter,
  type IParams,
} from "@workspace/ui/composed/pro-table/column-filter";
import { ColumnHeader } from "@workspace/ui/composed/pro-table/column-header";
import { ColumnToggle } from "@workspace/ui/composed/pro-table/column-toggle";
import { Pagination } from "@workspace/ui/composed/pro-table/pagination";
import { SortableRow } from "@workspace/ui/composed/pro-table/sortable-row";
import { ProTableWrapper } from "@workspace/ui/composed/pro-table/wrapper";
import { cn } from "@workspace/ui/lib/utils";
import {
  createFilterSnapshot,
  createPageRequest,
  normalizePageResult,
  type PageRequest,
  type PageResult,
} from "@workspace/ui/utils/pagination";
import { useSize } from "ahooks";
import { GripVertical, ListRestart, RefreshCcw } from "lucide-react";
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

export interface ProTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  request: (
    pagination: PageRequest,
    filter: TValue
  ) => Promise<PageResult<TData>>;
  params?: IParams[];
  header?: {
    title?: React.ReactNode;
    description?: React.ReactNode;
    toolbar?: React.ReactNode | React.ReactNode[];
    hidden?: boolean;
  };
  actions?: {
    render?: (row: TData) => React.ReactNode[];
    batchRender?: (rows: TData[]) => React.ReactNode[];
    columnSize?: number;
  };
  action?: React.Ref<ProTableActions | undefined>;
  texts?: Partial<{
    actions: string;
    asc: string;
    desc: string;
    hide: string;
    textRowsPerPage: string;
    textPageOf: (current: number, total: number) => string;
    selectedRowsText: (total: number) => string;
  }>;
  empty?: React.ReactNode;
  onSort?: (
    sourceId: string | number,
    targetId: string | number | null,
    items: TData[]
  ) => Promise<TData[]>;
  initialFilters?: Record<string, unknown>;
  onSelectionChange?: (rows: TData[]) => void;
}

export interface ProTableActions {
  refresh: () => void;
  reset: () => void;
  updateRow: (row: Record<string, unknown> & { id?: string | number }) => void;
}

interface ProTableActionMeta<TData> {
  render?: (row: TData) => React.ReactNode[];
}

export function ProTable<
  TData extends Record<string, unknown> & { id?: string | number },
  TValue extends Record<string, unknown>,
>({
  columns,
  request,
  params,
  header,
  actions,
  action,
  texts,
  empty,
  onSort,
  initialFilters,
  onSelectionChange,
}: ProTableProps<TData, TValue>) {
  const { t } = useTranslation("components");
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    if (initialFilters) {
      return Object.entries(initialFilters).map(([id, value]) => ({
        id,
        value,
      })) as ColumnFiltersState;
    }
    return [];
  });
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = useState({});
  const [data, setData] = useState<TData[]>([]);
  const [rowCount, setRowCount] = useState<number>(0);
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: 10,
  });
  const [loading, setLoading] = useState(false);
  const requestSequenceRef = useRef(0);
  const requestRef = useRef(request);

  useEffect(() => {
    requestRef.current = request;
  }, [request]);

  const filterSnapshot = useMemo(
    () => createFilterSnapshot<TValue>(columnFilters),
    [columnFilters]
  );
  const filterValues = filterSnapshot.values;
  const filterSignature = filterSnapshot.signature;
  const hasBatchActions = Boolean(actions?.batchRender);
  const hasRowActions = Boolean(actions?.render);
  const renderActionsCell = useCallback(
    ({ column, row }: CellContext<TData, TValue>) => {
      const meta = column.columnDef.meta as
        | ProTableActionMeta<TData>
        | undefined;
      return (
        <div className="flex items-center justify-end gap-2">
          {meta?.render?.(row.original)}
        </div>
      );
    },
    []
  );
  const tableColumns = useMemo(
    () =>
      [
        ...(onSort
          ? [
              {
                id: "sortable",
                header: (
                  <GripVertical className="h-4 w-4 cursor-move text-gray-500 hover:text-gray-700" />
                ),
                enableSorting: false,
                enableHiding: false,
                size: 34,
              },
            ]
          : []),
        ...(hasBatchActions
          ? [
              createSelectColumn<TData, TValue>({
                selectAll: t("table.selectAll", "Select all"),
                selectRow: t("table.selectRow", "Select row"),
              }),
            ]
          : []),
        ...columns.map(
          (column) =>
            ({
              enableSorting: false,
              ...column,
            }) as ColumnDef<TData, TValue>
        ),
        ...(hasRowActions
          ? ([
              {
                id: "actions",
                header: texts?.actions,
                cell: renderActionsCell,
                enableSorting: false,
                enableHiding: false,
                meta: {
                  render: actions?.render,
                },
                size: actions?.columnSize,
              },
            ] as ColumnDef<TData, TValue>[])
          : []),
      ] as ColumnDef<TData, TValue>[],
    [
      actions?.columnSize,
      actions?.render,
      columns,
      hasBatchActions,
      hasRowActions,
      onSort,
      renderActionsCell,
      t,
      texts?.actions,
    ]
  );

  const table = useReactTable({
    data,
    columns: tableColumns,
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
      pagination,
    },
    manualPagination: true,
    manualFiltering: true,
    rowCount,
    manualSorting: true,
    getRowId: getProTableRowId,
  });

  const fetchData = useCallback(async () => {
    const requestSequence = ++requestSequenceRef.current;
    setLoading(true);
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
    } finally {
      if (requestSequence === requestSequenceRef.current) {
        setLoading(false);
      }
    }
  }, [filterValues, pagination.pageIndex, pagination.pageSize]);
  const reset = useCallback(async () => {
    setSorting([]);
    setColumnFilters(
      initialFilters
        ? (Object.entries(initialFilters).map(([id, value]) => ({
            id,
            value,
          })) as ColumnFiltersState)
        : []
    );
    setColumnVisibility({});
    setRowSelection({});
    setPagination((previous) => ({ ...previous, pageIndex: 0 }));
  }, [initialFilters]);
  const ref = useRef<HTMLDivElement>(null);
  const size = useSize(ref);

  useImperativeHandle(
    action,
    () => ({
      refresh: fetchData,
      reset,
      updateRow: (row) => {
        setData((previous) =>
          previous.map((item) =>
            item.id === row.id ? { ...item, ...row } : item
          )
        );
      },
    }),
    [fetchData, reset]
  );

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pagination.pageIndex, pagination.pageSize, filterSignature]);

  const selectedRows = useMemo(
    () => table.getSelectedRowModel().flatRows.map((row) => row.original),
    [data, rowSelection, table]
  );
  const selectedCount = selectedRows.length;
  const hasExplicitColumnSizes = useMemo(
    () =>
      columnsHaveExplicitSizes(columns) ||
      typeof actions?.columnSize === "number",
    [actions?.columnSize, columns]
  );
  const explicitTableWidth = hasExplicitColumnSizes
    ? table
        .getVisibleLeafColumns()
        .reduce((width, column) => width + column.getSize(), 0)
    : undefined;

  useEffect(() => {
    onSelectionChange?.(selectedRows);
  }, [onSelectionChange, selectedRows]);

  return (
    <div className="flex flex-col gap-4" data-slot="pro-table" ref={ref}>
      {!header?.hidden && (
        <div className="flex flex-col gap-3" data-slot="pro-table-header">
          <div className="flex min-w-0 flex-wrap items-end justify-between gap-3">
            <div className="min-w-0">
              {header?.title ? (
                <div className="flex min-w-0 items-center gap-2">
                  <h2 className="truncate font-semibold text-base">
                    {header.title}
                  </h2>
                  {!loading ? (
                    <Badge variant="secondary">
                      {t("table.resultCount", "{{count}} items", {
                        count: rowCount,
                      })}
                    </Badge>
                  ) : null}
                </div>
              ) : null}
              {header?.description ? (
                <p className="mt-1 text-muted-foreground text-sm">
                  {header.description}
                </p>
              ) : null}
            </div>
            <div
              className="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2"
              data-slot="pro-table-actions"
            >
              {header?.toolbar}
              {header?.toolbar ? (
                <Separator
                  className="mx-0.5 hidden h-5 sm:block"
                  orientation="vertical"
                />
              ) : null}
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
              <ColumnToggle
                label={t("table.columns", "Columns")}
                table={table}
              />
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
            </div>
          </div>
          {params ? (
            <div className="min-w-0" data-slot="pro-table-filters">
              <ColumnFilter
                filters={filterValues}
                params={params}
                table={table}
              />
            </div>
          ) : null}
        </div>
      )}

      {selectedCount > 0 && actions?.batchRender && (
        <Alert className="flex items-center justify-between">
          <AlertTitle className="m-0">
            {texts?.selectedRowsText?.(selectedCount) ||
              t("table.selectedRows", "Selected {{count}} rows", {
                count: selectedCount,
              })}
          </AlertTitle>
          <AlertDescription className="flex gap-2">
            {actions.batchRender(selectedRows)}
          </AlertDescription>
        </Alert>
      )}

      <div
        className="relative w-auto overflow-x-auto rounded-lg border bg-card"
        data-slot="pro-table-viewport"
        style={{
          width: size?.width,
        }}
      >
        <ProTableWrapper data={data} onSort={onSort} setData={setData}>
          <Table
            className={cn("w-full", hasExplicitColumnSizes && "table-fixed")}
            style={
              explicitTableWidth ? { minWidth: explicitTableWidth } : undefined
            }
          >
            {hasExplicitColumnSizes ? (
              <colgroup>
                {table.getVisibleLeafColumns().map((column) => (
                  <col
                    key={column.id}
                    style={getColumnSizeStyle(column.columnDef.size)}
                  />
                ))}
              </colgroup>
            ) : null}
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      className={cn(
                        "!z-auto",
                        getTableHeaderClass(header.column.id)
                      )}
                      key={header.id}
                      style={getColumnSizeStyle(header.column.columnDef.size)}
                    >
                      <ColumnHeader
                        header={header}
                        text={{
                          asc: texts?.asc,
                          desc: texts?.desc,
                          hide: texts?.hide,
                        }}
                      />
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel()?.rows?.length ? (
                onSort ? (
                  table.getRowModel().rows.map((row) => (
                    <SortableRow
                      data-state={row.getIsSelected() && "selected"}
                      id={row.id}
                      isSortable
                      key={row.id}
                    >
                      {row
                        .getVisibleCells()
                        .filter((cell) => cell.column.id !== "sortable")
                        .map((cell) => (
                          <TableCell
                            className={getTableCellClass(cell.column.id)}
                            key={cell.id}
                            style={getColumnSizeStyle(
                              cell.column.columnDef.size
                            )}
                          >
                            {flexRender(
                              cell.column.columnDef.cell,
                              cell.getContext()
                            )}
                          </TableCell>
                        ))}
                    </SortableRow>
                  ))
                ) : (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      data-state={row.getIsSelected() && "selected"}
                      key={row.id}
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell
                          className={getTableCellClass(cell.column.id)}
                          key={cell.id}
                          style={getColumnSizeStyle(cell.column.columnDef.size)}
                        >
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                )
              ) : (
                <TableRow>
                  <TableCell className="py-16" colSpan={columns.length + 2}>
                    {empty || <Empty />}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </ProTableWrapper>

        {loading ? (
          <div className="absolute inset-0 z-20 flex items-center justify-center bg-card/80 backdrop-blur-[1px]">
            <Spinner className="text-primary" />
          </div>
        ) : null}
      </div>
      {rowCount > 0 && <Pagination table={table} />}
    </div>
  );
}

function getProTableRowId<TData extends { id?: string | number }>(
  row: TData,
  index: number
) {
  return row.id === undefined || row.id === null
    ? String(index)
    : String(row.id);
}

function columnsHaveExplicitSizes<TData, TValue>(
  columns: ColumnDef<TData, TValue>[]
): boolean {
  return columns.some((column) => {
    if (typeof column.size === "number") return true;
    if ("columns" in column && Array.isArray(column.columns)) {
      return columnsHaveExplicitSizes(column.columns);
    }
    return false;
  });
}

function createSelectColumn<TData, TValue>({
  selectAll,
  selectRow,
}: {
  selectAll: string;
  selectRow: string;
}): ColumnDef<TData, TValue> {
  return {
    id: "selected",
    header: ({ table }) => (
      <Checkbox
        aria-label={selectAll}
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && "indeterminate")
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        aria-label={selectRow}
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
      />
    ),
    enableSorting: false,
    enableHiding: false,
    size: 34,
  };
}

function getColumnSizeStyle(size?: number) {
  if (!(typeof size === "number" && Number.isFinite(size) && size > 0)) {
    return undefined;
  }
  return {
    width: size,
    minWidth: size,
    maxWidth: size,
  };
}

function getTableHeaderClass(columnId: string) {
  if (["sortable", "selected"].includes(columnId)) {
    return "bg-muted/70 md:sticky md:left-0 md:z-10 md:shadow-[2px_0_5px_-3px_rgba(0,0,0,0.12)] [&:has([role=checkbox])]:pr-2";
  }
  if (columnId === "actions") {
    return "bg-muted/70 text-right md:sticky md:right-0 md:z-10 md:shadow-[-2px_0_5px_-3px_rgba(0,0,0,0.12)]";
  }
  return "truncate";
}

function getTableCellClass(columnId: string) {
  if (["sortable", "selected"].includes(columnId)) {
    return "bg-card md:sticky md:left-0 md:shadow-[2px_0_5px_-3px_rgba(0,0,0,0.12)]";
  }
  if (columnId === "actions") {
    return "bg-card md:sticky md:right-0 md:shadow-[-2px_0_5px_-3px_rgba(0,0,0,0.12)]";
  }
  return "truncate";
}
