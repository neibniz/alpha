"use client";

import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
} from "@workspace/ui/components/chart";
import type { TrafficPoint, TrafficPeriod } from "@/lib/models";

const chartConfig = {
  downloadMbps: { color: "#2563eb", label: "下载" },
  uploadMbps: { color: "#737373", label: "上传" },
} satisfies ChartConfig;

function formatTick(timestamp: number, period: TrafficPeriod) {
  if (period === "7d" || period === "30d") {
    return new Intl.DateTimeFormat("zh-CN", {
      day: "2-digit",
      month: "2-digit",
    }).format(timestamp);
  }
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: period === "5m" || period === "15m" ? "2-digit" : undefined,
  }).format(timestamp);
}

function TooltipContent({
  active,
  label,
  payload,
}: Readonly<Record<string, unknown>>) {
  const values = payload as
    | Array<{ dataKey?: string; value?: number }>
    | undefined;
  if (!(active && values?.length)) return null;
  const upload =
    values.find((item) => item.dataKey === "uploadMbps")?.value || 0;
  const download =
    values.find((item) => item.dataKey === "downloadMbps")?.value || 0;
  return (
    <div className="grid min-w-44 gap-2 rounded-lg border bg-background p-3 text-xs shadow-lg">
      <time className="font-medium">
        {new Date(Number(label)).toLocaleTimeString("zh-CN")}
      </time>
      <span className="flex items-center gap-2">
        <i className="size-2 rounded-full bg-zinc-500" />
        上传 {upload.toFixed(2)} Mbps
      </span>
      <span className="flex items-center gap-2">
        <i className="size-2 rounded-full bg-blue-600" />
        下载 {download.toFixed(2)} Mbps
      </span>
    </div>
  );
}

export default function TrafficChart({
  period,
  points,
}: Readonly<{ period: TrafficPeriod; points: TrafficPoint[] }>) {
  return (
    <ChartContainer
      aria-label="实时流量趋势图"
      className="h-80 w-full aspect-auto"
      config={chartConfig}
      role="img"
    >
      <AreaChart
        data={points}
        margin={{ bottom: 4, left: 0, right: 8, top: 12 }}
      >
        <CartesianGrid stroke="#e9edef" vertical={false} />
        <XAxis
          axisLine={false}
          dataKey="timestamp"
          minTickGap={34}
          tick={{ fill: "#7a858a", fontSize: 12 }}
          tickFormatter={(value) => formatTick(Number(value), period)}
          tickLine={false}
          tickMargin={12}
        />
        <YAxis
          axisLine={false}
          domain={[0, "auto"]}
          tick={{ fill: "#7a858a", fontSize: 12 }}
          tickFormatter={(value) => `${Number(value).toFixed(0)} Mbps`}
          tickLine={false}
          width={70}
        />
        <ChartTooltip
          content={<TooltipContent />}
          cursor={{ stroke: "#cfd8dc", strokeDasharray: "3 3" }}
        />
        <Area
          dataKey="downloadMbps"
          dot={false}
          fill="var(--color-downloadMbps)"
          fillOpacity={0.08}
          isAnimationActive={period !== "5m" && period !== "15m"}
          stroke="var(--color-downloadMbps)"
          strokeWidth={2}
          type="monotone"
        />
        <Area
          dataKey="uploadMbps"
          dot={false}
          fill="var(--color-uploadMbps)"
          fillOpacity={0.06}
          isAnimationActive={period !== "5m" && period !== "15m"}
          stroke="var(--color-uploadMbps)"
          strokeWidth={2}
          type="monotone"
        />
      </AreaChart>
    </ChartContainer>
  );
}
