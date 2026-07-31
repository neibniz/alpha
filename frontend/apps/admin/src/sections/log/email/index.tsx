"use client";

import { useSearch } from "@tanstack/react-router";
import { Badge } from "@workspace/ui/components/badge";
import {
  ProTable,
  type ProTableActions,
} from "@workspace/ui/composed/pro-table/pro-table";
import { filterEmailLog } from "@workspace/ui/services/admin/log";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { formatDate } from "@/utils/common";
import {
  SensitiveLogToggle,
  sensitiveLogQuery,
  useSensitiveLogVisibility,
} from "../sensitive-log-toggle";

type EmailLogFilter = Pick<API.FilterEmailLogParams, "date" | "search">;

export default function EmailLogPage() {
  const { t } = useTranslation("log");
  const sp = useSearch({ strict: false }) as Record<string, string | undefined>;
  const tableAction = useRef<ProTableActions>(null);
  const { updateVisibility, visibility, visibilityRef } =
    useSensitiveLogVisibility(tableAction);

  const today = new Date().toISOString().split("T")[0];

  const initialFilters = {
    search: sp.search || undefined,
    date: sp.date || today,
  };
  return (
    <ProTable<API.AdminMessageLog, EmailLogFilter>
      action={tableAction}
      columns={[
        {
          accessorKey: "platform",
          header: t("column.platform", "Platform"),
          cell: ({ row }) => <Badge>{row.getValue("platform")}</Badge>,
        },
        { accessorKey: "to", header: t("column.to", "To") },
        { accessorKey: "subject", header: t("column.subject", "Subject") },
        {
          accessorKey: "content",
          header: t("column.content", "Content"),
          cell: ({ row }) => (
            <pre className="wrap-break-word max-w-[480px] overflow-auto whitespace-pre-wrap text-xs">
              {JSON.stringify(row.original.content || {}, null, 2)}
            </pre>
          ),
        },
        {
          accessorKey: "status",
          header: t("column.status", "Status"),
          cell: ({ row }) => {
            const status = row.original.status;
            const getStatusVariant = (status: number) => {
              if (status === 1) {
                return "default";
              }
              if (status === 0) {
                return "destructive";
              }
              return "outline";
            };

            const getStatusText = (status: number) => {
              if (status === 1) return t("sent", "Sent");
              if (status === 0) return t("failed", "Failed");
              return t("unknown", "Unknown");
            };

            return (
              <Badge variant={getStatusVariant(status)}>
                {getStatusText(status)}
              </Badge>
            );
          },
        },
        {
          accessorKey: "created_at",
          header: t("column.time", "Time"),
          cell: ({ row }) => formatDate(row.original.created_at),
        },
      ]}
      header={{
        title: t("title.email", "Email Log"),
        toolbar: (
          <SensitiveLogToggle onChange={updateVisibility} value={visibility} />
        ),
      }}
      initialFilters={initialFilters}
      params={[{ key: "search" }, { key: "date", type: "date" }]}
      request={async (pagination, filter) => {
        const { data } = await filterEmailLog({
          page: pagination.page,
          size: pagination.size,
          search: filter?.search,
          date: filter?.date,
          ...sensitiveLogQuery(visibilityRef.current),
        });
        return normalizePageResult(data?.data);
      }}
    />
  );
}
