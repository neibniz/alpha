"use client";

import { Link, useSearch } from "@tanstack/react-router";
import { Button } from "@workspace/ui/components/button";
import { ArrowUpRight } from "lucide-react";
import { ProTable } from "@/components/pro-table/pro-table";
import { filterUserSubscribeTrafficLog } from "@workspace/ui/services/admin/log";
import { formatBytes } from "@workspace/ui/utils/formatting";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useTranslation } from "react-i18next";
import { UserDetail, UserSubscribeDetail } from "@/sections/user/user-detail";

type SubscribeTrafficFilter = Pick<
  API.FilterUserSubscribeTrafficLogParams,
  "date" | "user_id" | "user_subscribe_id"
>;

export default function SubscribeTrafficLogPage() {
  const { t } = useTranslation("log");
  const sp = useSearch({ strict: false }) as Record<string, string | undefined>;

  const today = new Date().toISOString().split("T")[0];

  const initialFilters = {
    date: sp.date || today,
    user_id: sp.user_id ? Number(sp.user_id) : undefined,
    user_subscribe_id: sp.user_subscribe_id
      ? Number(sp.user_subscribe_id)
      : undefined,
  };
  return (
    <ProTable<API.UserSubscribeTrafficLog, SubscribeTrafficFilter>
      actions={{
        render: (row) => [
          <Button asChild key="detail" size="sm" variant="outline">
            <Link
              search={{
                date: row.date,
                user_id: row.user_id,
                subscribe_id: row.subscribe_id,
              }}
              to="/dashboard/log/traffic-details"
            >
              {t("detail", "Detail")}
              <ArrowUpRight />
            </Link>
          </Button>,
        ],
      }}
      columns={[
        {
          accessorKey: "user",
          header: t("column.user", "User"),
          cell: ({ row }) => <UserDetail id={Number(row.original.user_id)} />,
        },
        {
          accessorKey: "subscribe_id",
          header: t("column.subscribe", "Subscribe"),
          cell: ({ row }) => (
            <UserSubscribeDetail
              enabled
              hoverCard
              id={Number(row.original.subscribe_id)}
            />
          ),
        },
        {
          accessorKey: "upload",
          header: t("column.upload", "Upload"),
          cell: ({ row }) => formatBytes(row.original.upload),
        },
        {
          accessorKey: "download",
          header: t("column.download", "Download"),
          cell: ({ row }) => formatBytes(row.original.download),
        },
        {
          accessorKey: "total",
          header: t("column.total", "Total"),
          cell: ({ row }) => formatBytes(row.original.total),
        },
        {
          accessorKey: "date",
          header: t("column.date", "Date"),
        },
      ]}
      header={{
        title: t("title.subscribeTraffic", "Subscribe Traffic Log"),
        description: t(
          "subscribeTrafficDescription",
          "Review daily subscription usage and inspect individual traffic records."
        ),
      }}
      initialFilters={initialFilters}
      key={JSON.stringify(initialFilters)}
      params={[
        { key: "date", type: "date", placeholder: t("column.date", "Date") },
        { key: "user_id", placeholder: t("column.userId", "User ID") },
        {
          key: "user_subscribe_id",
          placeholder: t("column.subscribeId", "Subscribe ID"),
        },
      ]}
      request={async (pagination, filter) => {
        const { data } = await filterUserSubscribeTrafficLog({
          page: pagination.page,
          size: pagination.size,
          date: filter?.date,
          user_id: filter?.user_id,
          user_subscribe_id: filter?.user_subscribe_id,
        });
        return normalizePageResult(data?.data);
      }}
    />
  );
}
