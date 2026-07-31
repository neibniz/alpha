"use client";

import { useQuery } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@workspace/ui/components/chart";
import { Tabs, TabsList, TabsTrigger } from "@workspace/ui/components/tabs";
import { queryUserStatistics } from "@workspace/ui/services/admin/console";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Line, LineChart, XAxis, YAxis } from "recharts";
import { ChartNoAxesCombined } from "lucide-react";

type Period = "7d" | "30d" | "1y";

function localDateKey() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function MetricChip({
  label,
  tone,
  value,
}: {
  label: string;
  tone: "blue" | "cyan" | "green" | "orange";
  value?: number;
}) {
  const toneMap = {
    blue: "bg-blue-500",
    cyan: "bg-cyan-600",
    green: "bg-emerald-500",
    orange: "bg-orange-500",
  };
  return (
    <Badge className="h-7 min-w-0 gap-1.5 px-2.5 font-normal" variant="outline">
      <span className={`size-1.5 shrink-0 rounded-full ${toneMap[tone]}`} />
      <span className="truncate whitespace-nowrap text-muted-foreground">
        {label}
      </span>
      <div className="shrink-0 font-semibold tabular-nums">{value || 0}</div>
    </Badge>
  );
}

function EmptyChartState({ message }: { message: string }) {
  return (
    <Empty className="h-full border bg-muted/20 p-6">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <ChartNoAxesCombined />
        </EmptyMedia>
        <EmptyDescription>{message}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

export function UserStatisticsCard({
  realtime,
}: {
  realtime?: API.DashboardRealtimeResponse;
}) {
  const { t, i18n } = useTranslation("dashboard");
  const [period, setPeriod] = useState<Period>("7d");
  const { data: users } = useQuery({
    queryKey: ["queryUserStatistics", period],
    queryFn: async () => {
      const { data } = await queryUserStatistics({ period });
      return data.data;
    },
    refetchOnWindowFocus: true,
    staleTime: 60_000,
  });

  const todayRegister =
    realtime?.business?.register_today ?? users?.today?.register ?? 0;
  const todayNew =
    realtime?.business?.new_order_count ?? users?.today?.new_order_users ?? 0;
  const todayRenewal =
    realtime?.business?.renewal_order_count ??
    users?.today?.renewal_order_users ??
    0;
  const todayMembership =
    realtime?.business?.membership_order_count ??
    users?.today?.membership_users ??
    0;

  const chartData = useMemo(() => {
    const today = localDateKey();
    return (users?.series || []).map((item) => {
      const live =
        users?.granularity === "day" && item.date === today
          ? realtime?.business
          : undefined;
      return {
        date: item.date,
        membership: live?.membership_order_count ?? item.membership_users ?? 0,
        new_purchase: live?.new_order_count ?? item.new_order_users ?? 0,
        register: live?.register_today ?? item.register ?? 0,
        renewal: live?.renewal_order_count ?? item.renewal_order_users ?? 0,
      };
    });
  }, [realtime, users]);

  const hasData = chartData.some(
    (item) =>
      item.register + item.new_purchase + item.renewal + item.membership > 0
  );

  return (
    <Card className="self-start overflow-hidden">
      <CardHeader className="pb-0">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <CardTitle className="min-w-0 shrink truncate text-base">
                {t("userGrowth", "User Growth")}
              </CardTitle>
              <div className="flex h-8 shrink-0 items-center gap-2 rounded-md border border-border/60 bg-background px-3">
                <span className="whitespace-nowrap text-muted-foreground text-xs">
                  {t("todayRegister", "Today Register")}
                </span>
                <span className="whitespace-nowrap font-semibold text-sm tabular-nums">
                  {todayRegister}
                </span>
              </div>
            </div>
            <Tabs
              onValueChange={(value) => setPeriod(value as Period)}
              value={period}
            >
              <TabsList>
                <TabsTrigger value="7d">{t("last7Days", "7 days")}</TabsTrigger>
                <TabsTrigger value="30d">
                  {t("last30Days", "30 days")}
                </TabsTrigger>
                <TabsTrigger value="1y">{t("lastYear", "1 year")}</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
          <div className="flex min-w-0 flex-wrap gap-2">
            <MetricChip
              label={t("register", "Register")}
              tone="green"
              value={todayRegister}
            />
            <MetricChip
              label={t("membershipOpened", "Membership")}
              tone="cyan"
              value={todayMembership}
            />
            <MetricChip
              label={t("newPurchase", "New Purchase")}
              tone="blue"
              value={todayNew}
            />
            <MetricChip
              label={t("repurchase", "Repurchase")}
              tone="orange"
              value={todayRenewal}
            />
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-4">
        <div className="h-72">
          {hasData ? (
            <ChartContainer
              className="h-full w-full"
              config={{
                membership: {
                  label: t("membershipOpened", "Membership"),
                  color: "#22A69A",
                },
                new_purchase: {
                  label: t("newPurchase", "New Purchase"),
                  color: "#0A84FF",
                },
                register: {
                  label: t("register", "Register"),
                  color: "#34C759",
                },
                renewal: {
                  label: t("repurchase", "Repurchase"),
                  color: "#FF9F0A",
                },
              }}
            >
              <LineChart data={chartData} margin={{ left: 8, right: 12 }}>
                <XAxis
                  axisLine={false}
                  dataKey="date"
                  minTickGap={22}
                  tickFormatter={(value) =>
                    new Date(String(value)).toLocaleDateString(i18n.language, {
                      day: "numeric",
                      month: "short",
                    })
                  }
                  tickLine={false}
                  tickMargin={10}
                />
                <YAxis
                  allowDecimals={false}
                  axisLine={false}
                  tickLine={false}
                  width={28}
                />
                <ChartTooltip
                  content={<ChartTooltipContent />}
                  cursor={{ stroke: "hsl(var(--border))" }}
                />
                <Line
                  dataKey="register"
                  dot={false}
                  stroke="#34C759"
                  strokeWidth={2.2}
                  type="monotone"
                />
                <Line
                  dataKey="membership"
                  dot={false}
                  stroke="#22A69A"
                  strokeWidth={2.2}
                  type="monotone"
                />
                <Line
                  dataKey="new_purchase"
                  dot={false}
                  stroke="#0A84FF"
                  strokeWidth={2.2}
                  type="monotone"
                />
                <Line
                  dataKey="renewal"
                  dot={false}
                  stroke="#FF9F0A"
                  strokeWidth={2.2}
                  type="monotone"
                />
              </LineChart>
            </ChartContainer>
          ) : (
            <EmptyChartState
              message={t("empty.noAudienceData", "No audience data yet.")}
            />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
