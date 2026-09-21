"use client";

import { useSearch } from "@tanstack/react-router";
import { Badge } from "@workspace/ui/components/badge";
import { ProTable } from "@/components/pro-table/pro-table";
import { filterCommissionLog } from "@workspace/ui/services/admin/log";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useTranslation } from "react-i18next";
import { Display } from "@/components/display";
import { OrderLink } from "@/components/order-link";
import { UserDetail } from "@/sections/user/user-detail";
import { formatDate } from "@/utils/common";

type CommissionLogFilter = Pick<
  API.FilterCommissionLogParams,
  "date" | "user_id"
>;

export default function CommissionLogPage() {
  const { t } = useTranslation("log");
  const sp = useSearch({ strict: false }) as Record<string, string | undefined>;

  const today = new Date().toISOString().split("T")[0];

  const getCommissionTypeText = (type: number) => {
    const typeText = t(`type.${type}`, { defaultValue: "" });
    if (!typeText) {
      return `${t("unknown", "Unknown")} (${type})`;
    }
    return typeText;
  };

  const initialFilters = {
    date: sp.date || today,
    user_id: sp.user_id ? Number(sp.user_id) : undefined,
  };
  return (
    <ProTable<API.CommissionLog, CommissionLogFilter>
      columns={[
        {
          accessorKey: "user",
          header: t("column.user", "User"),
          cell: ({ row }) => <UserDetail id={Number(row.original.user_id)} />,
        },
        {
          accessorKey: "amount",
          header: t("column.amount", "Amount"),
          cell: ({ row }) => (
            <Display type="currency" value={row.original.amount} />
          ),
        },
        {
          accessorKey: "order_no",
          header: t("column.orderNo", "Order No."),
          cell: ({ row }) => <OrderLink orderId={row.original.order_no} />,
        },
        {
          accessorKey: "type",
          header: t("column.type", "Type"),
          cell: ({ row }) => (
            <Badge>{getCommissionTypeText(row.original.type)}</Badge>
          ),
        },
        {
          accessorKey: "timestamp",
          header: t("column.time", "Time"),
          cell: ({ row }) => formatDate(row.original.timestamp),
        },
      ]}
      header={{ title: t("title.commission", "Commission Log") }}
      initialFilters={initialFilters}
      key={JSON.stringify(initialFilters)}
      params={[
        { key: "date", type: "date" },
        { key: "user_id", placeholder: t("column.userId", "User ID") },
      ]}
      request={async (pagination, filter) => {
        const { data } = await filterCommissionLog({
          page: pagination.page,
          size: pagination.size,
          date: filter?.date,
          user_id: filter?.user_id,
        });
        return normalizePageResult(data?.data);
      }}
    />
  );
}
