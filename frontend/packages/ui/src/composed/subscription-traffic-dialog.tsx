"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@workspace/ui/components/chart";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { formatBytes } from "@workspace/ui/utils/formatting";
import { AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { useId, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import type {
  TrafficSeriesPeriod,
  TrafficSeriesResponse,
} from "../services/traffic-series";

const PERIODS: TrafficSeriesPeriod[] = ["5m", "15m", "1h", "24h", "7d", "30d"];

type SubscriptionTrafficDialogProps = {
  loadSeries: (
    period: TrafficSeriesPeriod
  ) => Promise<TrafficSeriesResponse | undefined>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  queryKey: readonly unknown[];
  title: string;
};

function formatTick(timestamp: number, period: TrafficSeriesPeriod) {
  if (["5m", "15m", "1h", "24h"].includes(period)) {
    return new Date(timestamp).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }
  return new Date(timestamp).toLocaleDateString([], {
    day: "numeric",
    month: "short",
  });
}

export function SubscriptionTrafficDialog({
  loadSeries,
  onOpenChange,
  open,
  queryKey,
  title,
}: Readonly<SubscriptionTrafficDialogProps>) {
  const { t } = useTranslation("user");
  const [period, setPeriod] = useState<TrafficSeriesPeriod>("24h");
  const chartId = useId().replaceAll(":", "");
  const downloadGradientId = `${chartId}-traffic-download`;
  const uploadGradientId = `${chartId}-traffic-upload`;
  const live = period === "5m" || period === "15m";
  const { data, isError, isFetching, refetch } = useQuery({
    enabled: open,
    queryKey: [...queryKey, period],
    queryFn: async () => {
      const response = await loadSeries(period);
      if (!response) {
        throw new Error("traffic series response is empty");
      }
      return response;
    },
    refetchInterval: live ? 5000 : false,
    refetchIntervalInBackground: false,
    staleTime: live ? 4000 : 30_000,
  });
  const hasTraffic = useMemo(
    () => data?.list.some((point) => point.download > 0 || point.upload > 0),
    [data?.list]
  );

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-[calc(100vw-1.5rem)] sm:max-w-3xl">
        <DialogHeader>
          <div className="flex flex-wrap items-center justify-between gap-3 pr-8">
            <DialogTitle>{title}</DialogTitle>
            <div className="flex items-center gap-1.5">
              <ToggleGroup
                aria-label={t("trafficPeriod", "Traffic period")}
                className="flex-wrap justify-end"
                onValueChange={(value) => {
                  if (value) setPeriod(value as TrafficSeriesPeriod);
                }}
                size="sm"
                type="single"
                value={period}
                variant="outline"
              >
                {PERIODS.map((value) => (
                  <ToggleGroupItem aria-label={value} key={value} value={value}>
                    {value}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    aria-label={t("refresh", "Refresh")}
                    disabled={isFetching}
                    onClick={() => refetch()}
                    size="icon-sm"
                    variant="outline"
                  >
                    <RefreshCw className={isFetching ? "animate-spin" : ""} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("refresh", "Refresh")}</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </DialogHeader>

        <div className="grid grid-cols-2 border-y">
          <div className="border-r px-3 py-2.5">
            <div className="text-muted-foreground text-xs">
              {t("download", "Download")}
            </div>
            <div className="mt-1 font-semibold text-base">
              {formatBytes(data?.download || 0)}
            </div>
          </div>
          <div className="px-3 py-2.5">
            <div className="text-muted-foreground text-xs">
              {t("upload", "Upload")}
            </div>
            <div className="mt-1 font-semibold text-base">
              {formatBytes(data?.upload || 0)}
            </div>
          </div>
        </div>

        <div className="relative h-72 min-w-0">
          {isFetching && !data ? (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/70">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : null}
          {isError ? (
            <div
              className="flex h-full items-center justify-center gap-2 text-destructive text-sm"
              role="alert"
            >
              <AlertCircle className="size-4" />
              {t("trafficLoadFailed", "Failed to load traffic usage")}
            </div>
          ) : null}
          {!isError && data ? (
            hasTraffic ? (
              <ChartContainer
                className="h-full w-full"
                config={{
                  download: {
                    color: "#0A84FF",
                    label: t("download", "Download"),
                  },
                  upload: {
                    color: "#30D158",
                    label: t("upload", "Upload"),
                  },
                }}
              >
                <AreaChart data={data.list} margin={{ left: 4, right: 12 }}>
                  <defs>
                    <linearGradient
                      id={downloadGradientId}
                      x1="0"
                      x2="0"
                      y1="0"
                      y2="1"
                    >
                      <stop offset="5%" stopColor="#0A84FF" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#0A84FF" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient
                      id={uploadGradientId}
                      x1="0"
                      x2="0"
                      y1="0"
                      y2="1"
                    >
                      <stop
                        offset="5%"
                        stopColor="#30D158"
                        stopOpacity={0.24}
                      />
                      <stop offset="95%" stopColor="#30D158" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis
                    axisLine={false}
                    dataKey="timestamp"
                    minTickGap={28}
                    tickFormatter={(value) => formatTick(Number(value), period)}
                    tickLine={false}
                    tickMargin={8}
                  />
                  <YAxis
                    axisLine={false}
                    tickFormatter={(value) => formatBytes(Number(value) || 0)}
                    tickLine={false}
                    width={68}
                  />
                  <ChartTooltip
                    content={
                      <ChartTooltipContent
                        formatter={(value, name) => [
                          formatBytes(Number(value) || 0),
                          name === "download"
                            ? t("download", "Download")
                            : t("upload", "Upload"),
                        ]}
                        labelFormatter={(value) =>
                          new Date(Number(value)).toLocaleString()
                        }
                      />
                    }
                    cursor={{ stroke: "hsl(var(--border))" }}
                  />
                  <Area
                    dataKey="download"
                    dot={false}
                    fill={`url(#${downloadGradientId})`}
                    isAnimationActive={!live}
                    stroke="#0A84FF"
                    strokeWidth={2}
                    type="monotone"
                  />
                  <Area
                    dataKey="upload"
                    dot={false}
                    fill={`url(#${uploadGradientId})`}
                    isAnimationActive={!live}
                    stroke="#30D158"
                    strokeWidth={2}
                    type="monotone"
                  />
                </AreaChart>
              </ChartContainer>
            ) : (
              <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                {t("trafficNoData", "No traffic usage in this period")}
              </div>
            )
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
