"use client";

import { useId, useSyncExternalStore } from "react";
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

const reducedMotionQuery = "(prefers-reduced-motion: reduce)";

function subscribeToMotionPreference(update: () => void) {
  const media = window.matchMedia(reducedMotionQuery);
  media.addEventListener("change", update);
  return () => media.removeEventListener("change", update);
}

function readReducedMotion() {
  return window.matchMedia(reducedMotionQuery).matches;
}

function reduceMotionOnServer() {
  return true;
}

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
  averagingSeconds,
  label,
  payload,
}: Readonly<{
  active?: boolean;
  averagingSeconds?: number;
  label?: number | string;
  payload?: Array<{ dataKey?: string; value?: number }>;
}>) {
  const values = payload;
  if (!(active && values?.length)) return null;
  const upload =
    values.find((item) => item.dataKey === "uploadMbps")?.value || 0;
  const download =
    values.find((item) => item.dataKey === "downloadMbps")?.value || 0;
  return (
    <div className="grid min-w-44 gap-2 rounded-lg border bg-background p-3 text-xs shadow-lg">
      <span className="flex items-center justify-between gap-4">
        <time className="font-medium">
          {new Date(Number(label)).toLocaleTimeString("zh-CN")}
        </time>
        {averagingSeconds ? (
          <span className="text-muted-foreground">
            {averagingSeconds} 秒均值
          </span>
        ) : null}
      </span>
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
  averagingSeconds,
  period,
  points,
}: Readonly<{
  averagingSeconds?: number;
  period: TrafficPeriod;
  points: TrafficPoint[];
}>) {
  const summaryId = useId();
  const reducedMotion = useSyncExternalStore(
    subscribeToMotionPreference,
    readReducedMotion,
    reduceMotionOnServer
  );
  if (points.length === 0) {
    return (
      <div
        className="flex h-80 items-center justify-center text-muted-foreground text-sm"
        role="status"
      >
        暂无流量记录
      </div>
    );
  }

  let peakDownload = 0;
  let peakUpload = 0;
  for (const point of points) {
    peakDownload = Math.max(peakDownload, point.downloadMbps);
    peakUpload = Math.max(peakUpload, point.uploadMbps);
  }
  const animate = !reducedMotion && period !== "5m" && period !== "15m";

  return (
    <figure className="min-w-0">
      <ChartContainer
        aria-describedby={summaryId}
        aria-label="流量速率趋势图"
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
            content={<TooltipContent averagingSeconds={averagingSeconds} />}
            cursor={{ stroke: "#cfd8dc", strokeDasharray: "3 3" }}
          />
          <Area
            dataKey="downloadMbps"
            dot={false}
            fill="var(--color-downloadMbps)"
            fillOpacity={0.08}
            isAnimationActive={animate}
            stroke="var(--color-downloadMbps)"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            type="monotoneX"
          />
          <Area
            dataKey="uploadMbps"
            dot={false}
            fill="var(--color-uploadMbps)"
            fillOpacity={0.06}
            isAnimationActive={animate}
            stroke="var(--color-uploadMbps)"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            type="monotoneX"
          />
        </AreaChart>
      </ChartContainer>
      <figcaption className="mt-2 text-muted-foreground text-sm" id={summaryId}>
        图中最高速率：下载 {peakDownload.toFixed(2)} Mbps，上传{" "}
        {peakUpload.toFixed(2)} Mbps。
      </figcaption>
    </figure>
  );
}
