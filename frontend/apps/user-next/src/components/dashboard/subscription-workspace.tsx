"use client";

import { useGSAP } from "@gsap/react";
import { useQuery } from "@tanstack/react-query";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@workspace/ui/components/empty";
import { Field, FieldLabel } from "@workspace/ui/components/field";
import { Progress } from "@workspace/ui/components/progress";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Separator } from "@workspace/ui/components/separator";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Textarea } from "@workspace/ui/components/textarea";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@workspace/ui/components/toggle-group";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import gsap from "gsap";
import {
  ArrowDown,
  ArrowUp,
  CalendarClock,
  Check,
  CircleAlert,
  Copy,
  Database,
  Gauge,
  Network,
  Plus,
  RefreshCcw,
  RefreshCw,
  RotateCcw,
  Settings2,
  ShoppingBag,
  Globe2,
} from "lucide-react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import {
  buildSubscriptionUrls,
  getSiteConfig,
  getSubscriptions,
  getSubscriptionScopes,
  getTrafficSeries,
} from "@/lib/client-api";
import { copyText } from "@/lib/copy-text";
import { formatBytes, formatDate, percentOf } from "@/lib/format";
import type {
  SiteConfig,
  Subscription,
  SubscriptionScope,
  TrafficPeriod,
} from "@/lib/models";
import { findDefaultPreset, presetKeyForSelection } from "@/lib/preset-utils";
import { PresetDialog } from "./preset-dialog";
import {
  type SubscriptionActionMode,
  SubscriptionActionsDialog,
} from "./subscription-actions-dialog";

gsap.registerPlugin(useGSAP);

const TrafficChart = dynamic(() => import("./traffic-chart"), {
  loading: () => (
    <div className="flex h-80 flex-col gap-3 p-4">
      <Skeleton className="h-full w-full rounded-lg" />
    </div>
  ),
  ssr: false,
});

const PERIOD_LABELS: Record<TrafficPeriod, string> = {
  "5m": "近 5 分钟",
  "15m": "近 15 分钟",
  "1h": "近 1 小时",
  "24h": "近 24 小时",
  "7d": "近 7 天",
  "30d": "近 30 天",
};

const STATUS_LABELS: Record<Subscription["status"], string> = {
  active: "正常",
  deducted: "已抵扣",
  exhausted: "流量已用尽",
  expired: "已到期",
  pending: "待生效",
  stopped: "已停用",
  unknown: "状态未知",
};

function SubscriptionRail({
  activeId,
  onSelect,
  subscriptions,
}: Readonly<{
  activeId: number;
  onSelect: (id: number) => void;
  subscriptions: Subscription[];
}>) {
  return (
    <section
      aria-label="订阅列表"
      className="flex items-stretch gap-3 lg:hidden"
      data-reveal
    >
      <div className="flex min-w-0 flex-1 snap-x gap-2 overflow-x-auto pb-1">
        {subscriptions.map((subscription) => {
          const used = subscription.uploadBytes + subscription.downloadBytes;
          const percent = percentOf(used, subscription.trafficBytes);
          const active = subscription.id === activeId;
          return (
            <Button
              aria-pressed={active}
              className={`alpha-subscription-stop h-auto min-w-[248px] snap-start flex-col items-stretch gap-3 rounded-lg border p-4 text-left shadow-none ${
                active
                  ? "border-primary bg-primary/5 text-foreground hover:bg-primary/7"
                  : "bg-card text-foreground hover:bg-muted/60"
              }`}
              key={subscription.id}
              onClick={() => onSelect(subscription.id)}
              type="button"
              variant="outline"
            >
              <span className="flex w-full items-center justify-between gap-3">
                <strong className="truncate text-sm">
                  {subscription.name}
                </strong>
                <Badge
                  className="shrink-0"
                  variant={
                    subscription.status === "active" ? "secondary" : "outline"
                  }
                >
                  {STATUS_LABELS[subscription.status]}
                </Badge>
              </span>
              <Progress aria-label="流量使用比例" value={percent} />
              <span className="flex w-full justify-between gap-3 text-muted-foreground text-xs">
                <span>
                  <b className="font-semibold text-foreground">
                    {formatBytes(used)}
                  </b>
                  {" / "}
                  {formatBytes(subscription.trafficBytes)}
                </span>
                <time>
                  {subscription.expiresAt
                    ? formatDate(subscription.expiresAt)
                    : "不限期"}
                </time>
              </span>
            </Button>
          );
        })}
      </div>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            asChild
            className="mt-1 shrink-0"
            size="icon"
            variant="outline"
          >
            <Link aria-label="购买新套餐" href="/products">
              <Plus aria-hidden="true" />
            </Link>
          </Button>
        </TooltipTrigger>
        <TooltipContent>购买新套餐</TooltipContent>
      </Tooltip>
    </section>
  );
}

function ActiveSubscription({
  config,
  scope,
  subscription,
}: Readonly<{
  config?: SiteConfig;
  scope?: SubscriptionScope;
  subscription: Subscription;
}>) {
  const root = useRef<HTMLDivElement>(null);
  const [period, setPeriod] = useState<TrafficPeriod>("5m");
  const [selectedPresetId, setSelectedPresetId] = useState<number>();
  const [presetOpen, setPresetOpen] = useState(false);
  const [actionMode, setActionMode] =
    useState<SubscriptionActionMode>("renewal");
  const [actionOpen, setActionOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [manualCopyOpen, setManualCopyOpen] = useState(false);
  const [selectedUrlIndex, setSelectedUrlIndex] = useState(0);
  const live = period === "5m" || period === "15m";
  const defaultPreset = findDefaultPreset(scope?.presets || []);
  const selectedPreset =
    selectedPresetId === undefined
      ? defaultPreset
      : scope?.presets.find((preset) => preset.id === selectedPresetId);
  const selectedAccessKeys = new Set(
    (selectedPreset?.items || []).map(
      (item) => `${item.nodeId}:${item.inboundAlias}`
    )
  );
  const activeNodes = selectedPreset
    ? (scope?.nodes || []).filter((node) => selectedAccessKeys.has(node.key))
    : scope?.nodes || [];
  const activeProtocols = [
    ...new Set(activeNodes.map((node) => node.protocol).filter(Boolean)),
  ];

  useEffect(() => {
    if (
      selectedPresetId !== undefined &&
      scope?.presets.some(
        (preset) => preset.id === selectedPresetId && preset.isDefault
      )
    ) {
      setSelectedPresetId(undefined);
      return;
    }
    if (
      selectedPresetId !== undefined &&
      !scope?.presets.some(
        (preset) => preset.id === selectedPresetId && preset.enabled
      )
    ) {
      setSelectedPresetId(undefined);
    }
  }, [scope?.presets, selectedPresetId]);

  const traffic = useQuery({
    queryFn: () => getTrafficSeries(subscription.id, period),
    queryKey: ["alpha-traffic", subscription.id, period],
    refetchInterval: (query) =>
      live
        ? Math.max(5000, (query.state.data?.intervalSeconds || 5) * 1000)
        : false,
    refetchIntervalInBackground: false,
    staleTime: live ? 4000 : 30_000,
  });

  useGSAP(
    () => {
      const targets = root.current?.querySelectorAll("[data-workbench-piece]");
      if (!targets?.length) return;
      const media = gsap.matchMedia();
      media.add("(prefers-reduced-motion: no-preference)", () => {
        gsap.fromTo(
          targets,
          { autoAlpha: 0, y: 12 },
          {
            autoAlpha: 1,
            duration: 0.55,
            ease: "power3.out",
            stagger: 0.055,
            y: 0,
          }
        );
      });
      return () => media.revert();
    },
    { dependencies: [subscription.id], revertOnUpdate: true }
  );

  const used = subscription.uploadBytes + subscription.downloadBytes;
  const usagePercent = percentOf(used, subscription.trafficBytes);
  const lastPoint = traffic.data?.points.at(-1);
  const links = buildSubscriptionUrls(
    subscription,
    config,
    presetKeyForSelection(selectedPresetId, selectedPreset)
  );
  const link = links[selectedUrlIndex] || links[0] || "";

  useEffect(() => {
    if (selectedUrlIndex < links.length) return;
    setSelectedUrlIndex(0);
  }, [links.length, selectedUrlIndex]);

  const copyLink = async () => {
    try {
      await copyText(link);
      setCopied(true);
      toast.success("订阅链接已复制");
      window.setTimeout(() => setCopied(false), 1800);
    } catch (error) {
      setManualCopyOpen(true);
      toast.info(
        error instanceof Error ? error.message : "请从弹窗中复制订阅链接"
      );
    }
  };

  return (
    <div className="flex flex-col gap-6" ref={root}>
      <section
        className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between"
        data-workbench-piece
      >
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="truncate font-semibold text-3xl tracking-tight">
              {subscription.name}
            </h1>
            <Badge className="gap-1.5" variant="secondary">
              <i
                aria-hidden="true"
                className={`size-1.5 rounded-full ${
                  subscription.status === "active"
                    ? "bg-green-600"
                    : "bg-zinc-400"
                }`}
              />
              {STATUS_LABELS[subscription.status]}
            </Badge>
          </div>
          <p className="mt-2 flex flex-wrap gap-x-2 gap-y-1 text-muted-foreground text-sm">
            <span>{formatBytes(subscription.trafficBytes)}</span>
            <span aria-hidden="true">·</span>
            <span>
              {subscription.speedLimitMbps
                ? `${subscription.speedLimitMbps} Mbps`
                : "不限速"}
            </span>
            <span aria-hidden="true">·</span>
            <span>{subscription.deviceLimit || "不限"} 台设备</span>
            <span aria-hidden="true">·</span>
            <span>
              到期{" "}
              {subscription.expiresAt
                ? formatDate(subscription.expiresAt)
                : "不限期"}
            </span>
          </p>
        </div>
        <Button className="w-full sm:w-auto" onClick={copyLink} type="button">
          {copied ? <Check /> : <Copy />}
          {copied ? "已复制" : "复制订阅链接"}
        </Button>
      </section>

      <section
        className="grid grid-cols-2 gap-3 xl:grid-cols-4"
        data-workbench-piece
      >
        <Card className="gap-4 rounded-lg py-5 shadow-none">
          <CardHeader className="grid grid-cols-[1fr_auto] items-center px-4 sm:px-5">
            <CardDescription>已用流量</CardDescription>
            <Gauge aria-hidden="true" className="text-muted-foreground" />
          </CardHeader>
          <CardContent className="grid gap-3 px-4 sm:px-5">
            <strong className="font-semibold text-xl tabular-nums sm:text-2xl">
              {formatBytes(used)}
              <small className="ml-1 font-normal text-muted-foreground text-xs sm:text-sm">
                / {formatBytes(subscription.trafficBytes)}
              </small>
            </strong>
            <Progress aria-label="流量使用比例" value={usagePercent} />
            <small className="text-muted-foreground">
              {usagePercent.toFixed(1)}%
            </small>
          </CardContent>
        </Card>
        <Card className="gap-4 rounded-lg py-5 shadow-none">
          <CardHeader className="grid grid-cols-[1fr_auto] items-center px-4 sm:px-5">
            <CardDescription>剩余流量</CardDescription>
            <Database aria-hidden="true" className="text-muted-foreground" />
          </CardHeader>
          <CardContent className="px-4 sm:px-5">
            <strong className="font-semibold text-xl tabular-nums sm:text-2xl">
              {formatBytes(subscription.remainingTrafficBytes)}
            </strong>
          </CardContent>
        </Card>
        <Card className="gap-4 rounded-lg py-5 shadow-none">
          <CardHeader className="grid grid-cols-[1fr_auto] items-center px-4 sm:px-5">
            <CardDescription>当前下载</CardDescription>
            <ArrowDown aria-hidden="true" className="text-blue-600" />
          </CardHeader>
          <CardContent className="px-4 sm:px-5">
            <strong className="font-semibold text-xl tabular-nums sm:text-2xl">
              {lastPoint?.downloadMbps.toFixed(1) || "0.0"}
              <small className="ml-1 font-normal text-muted-foreground text-sm">
                Mbps
              </small>
            </strong>
          </CardContent>
        </Card>
        <Card className="gap-4 rounded-lg py-5 shadow-none">
          <CardHeader className="grid grid-cols-[1fr_auto] items-center px-4 sm:px-5">
            <CardDescription>当前上传</CardDescription>
            <ArrowUp aria-hidden="true" className="text-zinc-500" />
          </CardHeader>
          <CardContent className="px-4 sm:px-5">
            <strong className="font-semibold text-xl tabular-nums sm:text-2xl">
              {lastPoint?.uploadMbps.toFixed(1) || "0.0"}
              <small className="ml-1 font-normal text-muted-foreground text-sm">
                Mbps
              </small>
            </strong>
          </CardContent>
        </Card>
      </section>

      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
        <Card
          className="overflow-hidden rounded-lg py-0 shadow-none"
          data-workbench-piece
        >
          <CardHeader className="gap-1 border-b px-5 py-5 has-data-[slot=card-action]:grid-cols-1 sm:px-6 xl:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
            <CardTitle className="flex items-center gap-3 text-base">
              实时流量
              <span className="flex items-center gap-1.5 font-normal text-muted-foreground text-xs">
                <i
                  className={`size-1.5 rounded-full ${live ? "bg-green-600" : "bg-zinc-400"}`}
                />
                {live ? "实时" : "历史"}
              </span>
            </CardTitle>
            <CardDescription>
              {PERIOD_LABELS[period]}的上下行速率
            </CardDescription>
            <CardAction className="col-span-1 col-start-1 row-span-1 row-start-3 mt-3 justify-self-stretch xl:col-start-2 xl:row-span-2 xl:row-start-1 xl:mt-0 xl:justify-self-end">
              <div className="flex flex-wrap items-center gap-2">
                <ToggleGroup
                  aria-label="流量时间范围"
                  className="flex-1"
                  onValueChange={(value) => {
                    if (value) setPeriod(value as TrafficPeriod);
                  }}
                  type="single"
                  value={period}
                  variant="outline"
                >
                  {(["5m", "15m", "1h", "24h", "7d"] as TrafficPeriod[]).map(
                    (value) => (
                      <ToggleGroupItem
                        className="px-2.5"
                        key={value}
                        value={value}
                      >
                        {value === "5m"
                          ? "5分钟"
                          : value === "15m"
                            ? "15分钟"
                            : value === "1h"
                              ? "1小时"
                              : value === "24h"
                                ? "24小时"
                                : "7天"}
                      </ToggleGroupItem>
                    )
                  )}
                </ToggleGroup>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      aria-label="刷新流量数据"
                      disabled={traffic.isFetching}
                      onClick={() => traffic.refetch()}
                      size="icon"
                      type="button"
                      variant="outline"
                    >
                      <RefreshCw
                        className={
                          traffic.isFetching ? "animate-spin" : undefined
                        }
                      />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>刷新流量数据</TooltipContent>
                </Tooltip>
              </div>
            </CardAction>
          </CardHeader>
          <CardContent className="p-0">
            <div className="p-3 sm:p-5" data-workbench-piece>
              {traffic.isError ? (
                <Alert
                  className="min-h-80 content-center"
                  variant="destructive"
                >
                  <CircleAlert />
                  <AlertTitle>流量数据加载失败</AlertTitle>
                  <AlertDescription>
                    <Button
                      className="h-auto px-0 py-0"
                      onClick={() => traffic.refetch()}
                      type="button"
                      variant="link"
                    >
                      重新加载
                    </Button>
                  </AlertDescription>
                </Alert>
              ) : (
                <TrafficChart
                  period={period}
                  points={traffic.data?.points || []}
                />
              )}
            </div>
            <div className="flex flex-wrap items-center gap-x-5 gap-y-2 border-t px-5 py-3 text-muted-foreground text-xs sm:px-6">
              <span className="flex items-center gap-2">
                <i className="size-2 rounded-full bg-blue-600" />
                下载
              </span>
              <span className="flex items-center gap-2">
                <i className="size-2 rounded-full bg-zinc-500" />
                上传
              </span>
              <time className="ml-auto">
                数据截至{" "}
                {lastPoint
                  ? new Date(lastPoint.timestamp).toLocaleString("zh-CN")
                  : "--"}
              </time>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-lg py-0 shadow-none" data-workbench-piece>
          <CardHeader className="border-b px-5 py-5">
            <CardTitle className="text-base">接入配置</CardTitle>
            <CardDescription>选择预设并复制订阅地址</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-5 px-5 py-5">
            <Field>
              <FieldLabel htmlFor={`preset-${subscription.id}`}>
                当前预设
              </FieldLabel>
              <Select
                onValueChange={(value) =>
                  setSelectedPresetId(
                    value === "default" ? undefined : Number(value)
                  )
                }
                value={
                  selectedPresetId === undefined
                    ? "default"
                    : String(selectedPresetId)
                }
              >
                <SelectTrigger
                  className="w-full"
                  id={`preset-${subscription.id}`}
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="default">
                      {defaultPreset
                        ? `默认 · ${defaultPreset.name}`
                        : "全部可用线路（默认）"}
                    </SelectItem>
                    {(scope?.presets || [])
                      .filter((preset) => !preset.isDefault)
                      .map((preset) => (
                        <SelectItem
                          disabled={!preset.enabled}
                          key={preset.id}
                          value={String(preset.id)}
                        >
                          {preset.name}
                          {preset.enabled ? "" : "（已停用）"}
                        </SelectItem>
                      ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <div>
              <span className="text-muted-foreground text-xs">协议组合</span>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {activeProtocols.length ? (
                  activeProtocols.map((protocol) => (
                    <Badge key={protocol} variant="secondary">
                      {protocol}
                    </Badge>
                  ))
                ) : (
                  <Badge variant="outline">暂无可用协议</Badge>
                )}
              </div>
            </div>
            {links.length > 1 ? (
              <Field>
                <FieldLabel htmlFor={`domain-${subscription.id}`}>
                  订阅域名
                </FieldLabel>
                <Select
                  onValueChange={(value) => setSelectedUrlIndex(Number(value))}
                  value={String(selectedUrlIndex)}
                >
                  <SelectTrigger
                    className="w-full"
                    id={`domain-${subscription.id}`}
                  >
                    <Globe2 />
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {links.map((item, index) => (
                        <SelectItem key={item} value={String(index)}>
                          {new URL(item).host}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            ) : null}
            <Separator />
            <dl className="grid gap-3 text-sm">
              <div className="flex items-center justify-between gap-4">
                <dt className="flex items-center gap-2 text-muted-foreground">
                  <Network />
                  入口数量
                </dt>
                <dd className="font-medium">{activeNodes.length} 个入口</dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt className="flex items-center gap-2 text-muted-foreground">
                  <CalendarClock />
                  下次重置
                </dt>
                <dd className="font-medium">
                  {subscription.resetAt
                    ? formatDate(subscription.resetAt)
                    : "按套餐周期"}
                </dd>
              </div>
            </dl>
          </CardContent>
          <CardFooter className="grid gap-2 border-t px-5 py-4">
            <Button
              className="w-full"
              onClick={() => setPresetOpen(true)}
              type="button"
              variant="outline"
            >
              <Settings2 />
              管理预设
            </Button>
            {subscription.allowRenewal ? (
              <Button
                className="w-full"
                disabled={!subscription.canRenew}
                onClick={() => {
                  setActionMode("renewal");
                  setActionOpen(true);
                }}
                title={
                  subscription.canRenew
                    ? "续费订阅"
                    : subscription.renewalReason
                }
                type="button"
                variant="outline"
              >
                <RotateCcw />
                续费订阅
              </Button>
            ) : null}
            {subscription.allowResetTraffic ? (
              <Button
                className="w-full"
                disabled={!subscription.canResetTraffic}
                onClick={() => {
                  setActionMode("reset");
                  setActionOpen(true);
                }}
                title={
                  subscription.canResetTraffic
                    ? "重置流量"
                    : subscription.resetTrafficReason
                }
                type="button"
                variant="outline"
              >
                <RefreshCcw />
                重置流量
              </Button>
            ) : null}
          </CardFooter>
        </Card>
      </div>

      <PresetDialog
        config={config}
        initialPresetId={selectedPresetId ?? defaultPreset?.id}
        nodes={scope?.nodes || []}
        onOpenChange={setPresetOpen}
        onPresetSelected={setSelectedPresetId}
        open={presetOpen}
        presets={scope?.presets || []}
        subscription={subscription}
      />
      <SubscriptionActionsDialog
        config={config}
        mode={actionMode}
        onOpenChange={setActionOpen}
        open={actionOpen}
        subscription={subscription}
      />
      <Dialog onOpenChange={setManualCopyOpen} open={manualCopyOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>复制订阅链接</DialogTitle>
            <DialogDescription>
              浏览器未授予剪贴板权限，完整链接已在下方准备好。
            </DialogDescription>
          </DialogHeader>
          <Field>
            <FieldLabel htmlFor={`subscription-link-${subscription.id}`}>
              订阅链接
            </FieldLabel>
            <Textarea
              id={`subscription-link-${subscription.id}`}
              className="min-h-28 resize-none font-mono text-xs"
              onFocus={(event) => event.currentTarget.select()}
              readOnly
              value={link}
            />
          </Field>
          <DialogFooter>
            <Button
              onClick={() => setManualCopyOpen(false)}
              type="button"
              variant="outline"
            >
              完成
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export function SubscriptionWorkspace() {
  const root = useRef<HTMLDivElement>(null);
  const searchParams = useSearchParams();
  const requestedId = Number(searchParams.get("subscription")) || 0;
  const [selectedId, setSelectedId] = useState(requestedId);
  const subscriptions = useQuery({
    queryFn: getSubscriptions,
    queryKey: ["alpha-subscriptions"],
  });
  const scopes = useQuery({
    queryFn: getSubscriptionScopes,
    queryKey: ["alpha-subscription-scopes"],
  });
  const siteConfig = useQuery({
    queryFn: getSiteConfig,
    queryKey: ["alpha-site-config"],
    staleTime: 300_000,
  });

  useEffect(() => {
    if (!subscriptions.data?.length) return;
    const nextId = subscriptions.data.some((item) => item.id === requestedId)
      ? requestedId
      : subscriptions.data[0]?.id;
    if (nextId && nextId !== selectedId) setSelectedId(nextId);
  }, [requestedId, selectedId, subscriptions.data]);

  useGSAP(
    () => {
      const targets = root.current?.querySelectorAll(
        ".alpha-subscription-stop"
      );
      if (!targets?.length) return;
      const media = gsap.matchMedia();
      media.add("(prefers-reduced-motion: no-preference)", () => {
        gsap.fromTo(
          targets,
          { autoAlpha: 0, y: -10 },
          {
            autoAlpha: 1,
            duration: 0.5,
            ease: "power3.out",
            stagger: 0.07,
            y: 0,
          }
        );
      });
      return () => media.revert();
    },
    {
      dependencies: [subscriptions.data?.length],
      revertOnUpdate: true,
    }
  );

  if (subscriptions.isLoading) {
    return (
      <div className="mx-auto flex w-full max-w-[1600px] flex-col gap-4 px-4 py-5 sm:px-6 lg:px-8 lg:py-7">
        <Skeleton className="h-28 w-full rounded-lg" />
        <Skeleton className="h-[520px] w-full rounded-lg" />
      </div>
    );
  }
  if (subscriptions.isError) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6">
        <Alert variant="destructive">
          <CircleAlert />
          <AlertTitle>订阅加载失败</AlertTitle>
          <AlertDescription>
            <Button
              className="h-auto px-0 py-0"
              onClick={() => subscriptions.refetch()}
              type="button"
              variant="link"
            >
              重新加载
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }
  if (!subscriptions.data?.length) {
    return (
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-3xl px-4 py-12 sm:px-6">
        <Empty className="w-full border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ShoppingBag aria-hidden="true" />
            </EmptyMedia>
            <EmptyTitle>还没有订阅</EmptyTitle>
            <EmptyDescription>
              选择套餐后，实时流量与接入配置会集中显示在这里。
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button asChild>
              <Link href="/products">浏览套餐</Link>
            </Button>
          </EmptyContent>
        </Empty>
      </div>
    );
  }

  const selected =
    subscriptions.data.find((item) => item.id === selectedId) ||
    subscriptions.data[0];
  if (!selected) return null;
  const scope = scopes.data?.find((item) => item.id === selected.id);

  return (
    <div
      className="mx-auto flex w-full max-w-[1500px] flex-col gap-6 px-4 py-6 sm:px-6 lg:px-7 lg:py-8"
      ref={root}
    >
      <SubscriptionRail
        activeId={selected.id}
        onSelect={setSelectedId}
        subscriptions={subscriptions.data}
      />
      <ActiveSubscription
        config={siteConfig.data}
        key={selected.id}
        scope={scope}
        subscription={selected}
      />
      {scopes.isError ? (
        <Alert variant="destructive">
          <CircleAlert />
          <AlertTitle>接入预设加载失败</AlertTitle>
          <AlertDescription className="sm:grid-cols-[1fr_auto] sm:items-center">
            <span>流量与订阅操作仍可使用。</span>
            <Button
              className="h-auto px-0 py-0"
              onClick={() => scopes.refetch()}
              type="button"
              variant="link"
            >
              重新加载
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
