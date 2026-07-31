"use client";

import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Input } from "@workspace/ui/components/input";
import { Label } from "@workspace/ui/components/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
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
  TooltipProvider,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import Empty from "@workspace/ui/composed/empty";
import { filterProxyAccessLog } from "@workspace/ui/services/admin/log";
import {
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Loader2,
  RefreshCw,
  RotateCcw,
  Search,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { UserDetail, UserSubscribeDetail } from "@/sections/user/user-detail";
import { useServer } from "@/stores/server";
import { formatDate } from "@/utils/common";
import {
  proxyAccessFilterError,
  proxyAccessFilterParams,
  type ProxyAccessFilters,
} from "./filter-data";

const pageSize = 50;

function toLocalInputValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function initialFilters(): ProxyAccessFilters {
  const end = new Date();
  return {
    start: toLocalInputValue(new Date(end.getTime() - 24 * 60 * 60 * 1000)),
    end: toLocalInputValue(end),
    serverId: "",
    userId: "",
    subscribeId: "",
    sourceIP: "",
    destination: "",
    status: "all",
  };
}

export default function ProxyAccessLogPage() {
  const { t } = useTranslation("log");
  const { getServerName } = useServer();
  const initial = useMemo(initialFilters, []);
  const [draft, setDraft] = useState<ProxyAccessFilters>(initial);
  const [filters, setFilters] = useState<ProxyAccessFilters>(initial);
  const [rows, setRows] = useState<API.ProxyAccessLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [cursorHistory, setCursorHistory] = useState<string[]>([""]);
  const [pageIndex, setPageIndex] = useState(0);
  const [nextCursor, setNextCursor] = useState("");
  const requestID = useRef(0);
  const draftError = useMemo(() => proxyAccessFilterError(draft), [draft]);

  const filterErrorMessage = draftError
    ? t(`filterError.${draftError}`, {
        defaultValue: {
          invalidTime: "请选择有效的开始和结束时间",
          invalidTimeOrder: "开始时间必须早于结束时间",
          timeRangeTooLarge: "单次查询时间范围不能超过 31 天",
          invalidServerId: "服务器 ID 必须是正整数",
          invalidUserId: "用户 ID 必须是正整数",
          invalidSubscribeId: "订阅 ID 必须是正整数",
        }[draftError],
      })
    : "";

  const load = useCallback(async () => {
    const currentRequest = ++requestID.current;
    setLoading(true);
    setLoadFailed(false);
    try {
      const { data } = await filterProxyAccessLog(
        proxyAccessFilterParams(filters, pageSize, cursorHistory[pageIndex])
      );
      if (currentRequest !== requestID.current) return;
      setRows(data?.data?.list || []);
      setNextCursor(data?.data?.next_cursor || "");
    } catch {
      if (currentRequest !== requestID.current) return;
      setRows([]);
      setNextCursor("");
      setLoadFailed(true);
    } finally {
      if (currentRequest === requestID.current) setLoading(false);
    }
  }, [cursorHistory, filters, pageIndex]);

  useEffect(() => {
    load();
  }, [load]);

  const applyFilters = () => {
    if (draftError) return;
    setCursorHistory([""]);
    setPageIndex(0);
    setFilters({ ...draft });
  };

  const resetFilters = () => {
    const reset = initialFilters();
    setDraft(reset);
    setCursorHistory([""]);
    setPageIndex(0);
    setFilters(reset);
  };

  const goNext = () => {
    if (!nextCursor) return;
    setCursorHistory((current) => {
      const next = current.slice(0, pageIndex + 1);
      next.push(nextCursor);
      return next;
    });
    setPageIndex((current) => current + 1);
  };

  return (
    <AdminPage>
      <AdminPageHeader
        actions={
          <div className="flex items-center gap-2">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    aria-label={t("refresh", "Refresh")}
                    disabled={loading}
                    onClick={load}
                    size="icon-sm"
                    variant="outline"
                  >
                    <RefreshCw />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("refresh", "Refresh")}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <Button disabled={loading} onClick={resetFilters} variant="outline">
              <RotateCcw />
              {t("proxyAccess.reset", "Reset")}
            </Button>
            <Button
              disabled={loading || Boolean(draftError)}
              onClick={applyFilters}
            >
              {loading ? <Loader2 className="animate-spin" /> : <Search />}
              {t("search", "Search")}
            </Button>
          </div>
        }
        description={t(
          "proxyAccess.description",
          "Query privacy-protected Xray access events stored in ClickHouse"
        )}
        eyebrow={t("workspace", "Analytics")}
        title={t("title.proxyAccess", "Proxy Access")}
      />

      <div className="border-y py-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-start">
              {t("filter.startTime", "Start time")}
            </Label>
            <Input
              aria-invalid={
                draftError === "invalidTime" ||
                draftError === "invalidTimeOrder" ||
                draftError === "timeRangeTooLarge"
              }
              id="proxy-access-start"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  start: event.target.value,
                }))
              }
              type="datetime-local"
              value={draft.start}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-end">
              {t("filter.endTime", "End time")}
            </Label>
            <Input
              aria-invalid={
                draftError === "invalidTime" ||
                draftError === "invalidTimeOrder" ||
                draftError === "timeRangeTooLarge"
              }
              id="proxy-access-end"
              onChange={(event) =>
                setDraft((current) => ({ ...current, end: event.target.value }))
              }
              type="datetime-local"
              value={draft.end}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-server">
              {t("column.serverId", "Server ID")}
            </Label>
            <Input
              aria-invalid={draftError === "invalidServerId"}
              id="proxy-access-server"
              inputMode="numeric"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  serverId: event.target.value,
                }))
              }
              placeholder={t("proxyAccess.optional", "Optional")}
              value={draft.serverId}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-user">
              {t("column.userId", "User ID")}
            </Label>
            <Input
              aria-invalid={draftError === "invalidUserId"}
              id="proxy-access-user"
              inputMode="numeric"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  userId: event.target.value,
                }))
              }
              placeholder={t("proxyAccess.optional", "Optional")}
              value={draft.userId}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-subscribe">
              {t("column.subscribeId", "Subscribe ID")}
            </Label>
            <Input
              aria-invalid={draftError === "invalidSubscribeId"}
              id="proxy-access-subscribe"
              inputMode="numeric"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  subscribeId: event.target.value,
                }))
              }
              placeholder={t("proxyAccess.optional", "Optional")}
              value={draft.subscribeId}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-source">
              {t("column.sourceIp", "Source IP")}
            </Label>
            <Input
              id="proxy-access-source"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  sourceIP: event.target.value,
                }))
              }
              placeholder={t("proxyAccess.exactMatch", "Exact match")}
              value={draft.sourceIP}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="proxy-access-destination">
              {t("column.destination", "Destination")}
            </Label>
            <Input
              id="proxy-access-destination"
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  destination: event.target.value,
                }))
              }
              placeholder={t("proxyAccess.exactMatch", "Exact match")}
              value={draft.destination}
            />
          </div>
          <div className="space-y-1.5">
            <Label>{t("column.status", "Status")}</Label>
            <Select
              onValueChange={(value) =>
                setDraft((current) => ({ ...current, status: value }))
              }
              value={draft.status}
            >
              <SelectTrigger aria-label={t("column.status", "Status")}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="all">
                    {t("status.all", "All statuses")}
                  </SelectItem>
                  <SelectItem value="accepted">
                    {t("status.accepted", "Accepted")}
                  </SelectItem>
                  <SelectItem value="rejected">
                    {t("status.rejected", "Rejected")}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>
        {filterErrorMessage ? (
          <p
            className="mt-3 flex items-center gap-2 text-destructive text-sm"
            role="alert"
          >
            <AlertCircle className="size-4" />
            {filterErrorMessage}
          </p>
        ) : (
          <p className="mt-3 text-muted-foreground text-xs">
            {t(
              "proxyAccess.privacyHint",
              "IP addresses and destinations are privacy-masked in results; exact-match filters use protected indexes"
            )}
          </p>
        )}
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="overflow-x-auto">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow>
                <TableHead>{t("column.time", "Time")}</TableHead>
                <TableHead>{t("column.server", "Server")}</TableHead>
                <TableHead>{t("column.user", "User")}</TableHead>
                <TableHead>{t("column.subscribe", "Subscribe")}</TableHead>
                <TableHead>{t("column.source", "Source")}</TableHead>
                <TableHead>{t("column.destination", "Destination")}</TableHead>
                <TableHead>{t("column.network", "Network")}</TableHead>
                <TableHead>{t("column.status", "Status")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.event_id}>
                  <TableCell className="whitespace-nowrap">
                    {formatDate(row.timestamp)}
                  </TableCell>
                  <TableCell className="whitespace-nowrap">
                    {row.server_id
                      ? `${getServerName(row.server_id)} (${row.server_id})`
                      : "--"}
                  </TableCell>
                  <TableCell>
                    {row.user_id ? <UserDetail id={row.user_id} /> : "--"}
                  </TableCell>
                  <TableCell>
                    {row.user_subscribe_id ? (
                      <UserSubscribeDetail
                        enabled
                        hoverCard
                        id={row.user_subscribe_id}
                      />
                    ) : (
                      "--"
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {row.source_ip || "--"}
                    {row.source_port ? `:${row.source_port}` : ""}
                  </TableCell>
                  <TableCell className="max-w-64 font-mono text-xs">
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="block truncate">
                            {row.destination || "--"}
                            {row.destination_port
                              ? `:${row.destination_port}`
                              : ""}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          {row.reason || row.destination || "--"}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>
                  <TableCell className="uppercase">
                    {row.network || "--"}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        row.status === "accepted" ? "default" : "destructive"
                      }
                    >
                      {t(`status.${row.status}`, row.status)}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        {!loading && loadFailed ? (
          <div
            className="flex min-h-28 items-center justify-center gap-2 border-t text-muted-foreground text-sm"
            role="alert"
          >
            <AlertCircle className="size-4" />
            {t("loadFailed", "Failed to load access logs")}
          </div>
        ) : null}
        {!(loading || loadFailed) && rows.length === 0 ? (
          <div className="border-t">
            <Empty />
          </div>
        ) : null}
      </div>

      <div className="flex items-center justify-end gap-2">
        <Button
          aria-label={t("pagination.previous", "Previous page")}
          disabled={loading || pageIndex === 0}
          onClick={() => setPageIndex((current) => Math.max(0, current - 1))}
          size="icon-sm"
          variant="outline"
        >
          <ChevronLeft />
        </Button>
        <span className="min-w-12 text-center text-muted-foreground text-sm">
          {pageIndex + 1}
        </span>
        <Button
          aria-label={t("pagination.next", "Next page")}
          disabled={loading || !nextCursor}
          onClick={goNext}
          size="icon-sm"
          variant="outline"
        >
          <ChevronRight />
        </Button>
      </div>
    </AdminPage>
  );
}
