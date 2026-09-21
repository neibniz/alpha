"use client";

import { useSearch } from "@tanstack/react-router";
import { Badge } from "@workspace/ui/components/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import { filterLoginLog } from "@workspace/ui/services/admin/log";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { IpLink, IpLocation } from "@/components/ip-link";
import { UserDetail } from "@/sections/user/user-detail";
import { formatDate } from "@/utils/common";
import {
  SensitiveLogToggle,
  sensitiveLogQuery,
  useSensitiveLogVisibility,
} from "../sensitive-log-toggle";

type LoginLogFilter = Pick<API.FilterLoginLogParams, "date" | "user_id">;

export default function LoginLogPage() {
  const { t } = useTranslation("log");
  const sp = useSearch({ strict: false }) as Record<string, string | undefined>;
  const tableAction = useRef<ProTableActions>(null);
  const { updateVisibility, visibility, visibilityRef } =
    useSensitiveLogVisibility(tableAction);

  const today = new Date().toISOString().split("T")[0];

  const initialFilters = {
    date: sp.date || today,
    user_id: sp.user_id ? Number(sp.user_id) : undefined,
  };
  return (
    <ProTable<API.LoginLog, LoginLogFilter>
      action={tableAction}
      columns={[
        {
          accessorKey: "user",
          header: t("column.user", "User"),
          cell: ({ row }) => (
            <div>
              <Badge className="capitalize">{row.original.method}</Badge>{" "}
              <UserDetail id={Number(row.original.user_id)} />
            </div>
          ),
        },

        {
          accessorKey: "login_ip",
          header: t("column.ip", "IP"),
          cell: ({ row }) => (
            <IpLink ip={String(row.original.login_ip || "")} />
          ),
        },
        {
          accessorKey: "location",
          header: t("column.location", "Location"),
          cell: ({ row }) => <IpLocation location={row.original.location} />,
        },
        {
          accessorKey: "user_agent",
          header: t("column.userAgent", "User Agent"),
          cell: ({ row }) => {
            const userAgent = String(row.original.user_agent || "");
            return (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="max-w-48 cursor-help truncate">
                      {userAgent}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className="wrap-break-word max-w-md">{userAgent}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            );
          },
        },
        {
          accessorKey: "success",
          header: t("column.success", "Success"),
          cell: ({ row }) => (
            <Badge variant={row.original.success ? "default" : "destructive"}>
              {row.original.success
                ? t("success", "Success")
                : t("failed", "Failed")}
            </Badge>
          ),
        },
        {
          accessorKey: "timestamp",
          header: t("column.time", "Time"),
          cell: ({ row }) => formatDate(row.original.timestamp),
        },
      ]}
      header={{
        title: t("title.login", "Login Log"),
        toolbar: (
          <SensitiveLogToggle onChange={updateVisibility} value={visibility} />
        ),
      }}
      initialFilters={initialFilters}
      key={JSON.stringify(initialFilters)}
      params={[
        { key: "date", type: "date" },
        { key: "user_id", placeholder: t("column.userId", "User ID") },
      ]}
      request={async (pagination, filter) => {
        const { data } = await filterLoginLog({
          page: pagination.page,
          size: pagination.size,
          date: filter?.date,
          user_id: filter?.user_id,
          ...sensitiveLogQuery(visibilityRef.current),
        });
        return normalizePageResult(data?.data);
      }}
    />
  );
}
